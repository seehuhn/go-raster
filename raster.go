// seehuhn.de/go/raster - a 2D rendering library
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package raster

import (
	"cmp"
	"math"
	"slices"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// edge represents a line segment in device coordinates.
type edge struct {
	x0, y0 float64 // start point
	x1, y1 float64 // end point
	dxdy   float64 // (x1-x0)/(y1-y0), precomputed for x-intercept calculation
}

// Rasterizer converts vector paths to pixel coverage values—the fraction of
// each pixel's area covered by the filled/stroked path, ranging from 0
// (outside) to 1 (inside). Create one instance and reuse it for multiple
// paths. Internal buffers grow as needed but never shrink, achieving zero
// allocations in steady state.
//
// A Rasterizer is not safe for concurrent use.
type Rasterizer struct {
	// CTM transforms from user space to device space. Must be non-singular.
	CTM matrix.Matrix

	// Clip bounds output to this device-coordinate rectangle.
	// Coordinates must be integer-aligned.
	Clip rect.Rect

	// Flatness controls curve approximation accuracy in device pixels.
	// Typical values: 0.25–1.0. Must be positive.
	Flatness float64

	// Width sets stroke thickness in user-space units.
	// Must be positive for stroke operations.
	Width float64

	// Cap sets the style for stroke endpoints (butt, round, or square).
	Cap graphics.LineCapStyle

	// Join sets the style for stroke corners (miter, round, or bevel).
	Join graphics.LineJoinStyle

	// MiterLimit caps miter join length. Must be at least 1.0.
	MiterLimit float64

	// Dash specifies alternating on/off lengths in user-space units.
	// All elements must be non-negative, and at least one must be positive.
	// Nil means solid (no dashing).
	Dash []float64

	// DashPhase offsets into the dash pattern in user-space units.
	// Can be any value (positive, negative, or zero).
	DashPhase float64

	// smallPathThreshold is the maximum bounding box area (in pixels) for
	// using 2D buffers (Approach A). Paths with larger bounding boxes use
	// the active edge list (Approach B).
	smallPathThreshold int

	// Internal buffers (reused across calls)
	cover         []float32  // coverage accumulation: cover change per pixel; reused as output
	area          []float32  // coverage accumulation: area within pixel
	edges         []edge     // edge list for current path (device coordinates)
	activeIdx     []int      // indices of active edges
	rowHasEdges   []bool     // per-scanline flag: true if any edge contributes
	stroke        []vec.Vec2 // stroke outline vertices (all subpaths contiguous)
	strokeOffsets []int      // start index of each stroke polygon in stroke[]

	// Flattening buffers (for stroke path processing)
	segs             []strokeSegment // all segments from all subpaths, contiguous
	segsOffsets      []int           // start index of each subpath in segments
	subpathClosed    []bool          // whether each subpath is closed
	degeneratePoints []vec.Vec2      // degenerate subpaths (no orientation)

	// Edge collection state (used by collectEdges/addEdge)
	edgeBBoxFirst bool    // true if no edges added yet
	edgeDevXMin   float64 // bounding box in device space
	edgeDevXMax   float64
	edgeDevYMin   float64
	edgeDevYMax   float64

	// Dash pattern output buffers
	dashedSegs        []strokeSegment // all dashed segments, contiguous
	dashedSegsOffsets []int           // start index of each dashed subpath
}

// NewRasterizer returns a Rasterizer with the given clip rectangle and
// PDF default values for other parameters.
func NewRasterizer(clip rect.Rect) *Rasterizer {
	return &Rasterizer{
		CTM:        matrix.Identity,
		Clip:       clip,
		Flatness:   defaultFlatness,
		Width:      1.0,
		Cap:        graphics.LineCapButt,
		Join:       graphics.LineJoinMiter,
		MiterLimit: defaultMiterLimit,

		smallPathThreshold: smallPathThreshold,
	}
}

// transformLinear applies only the 2×2 linear part of CTM to a vector.
// Used for CTM-aware tolerance checking where translation is irrelevant.
func (r *Rasterizer) transformLinear(v vec.Vec2) vec.Vec2 {
	return vec.Vec2{
		X: r.CTM[0]*v.X + r.CTM[2]*v.Y,
		Y: r.CTM[1]*v.X + r.CTM[3]*v.Y,
	}
}

// flattenQuadratic flattens a quadratic Bézier and calls emit for each line segment.
// p0 is the start point (current point), p1 is control, p2 is endpoint.
// All points are in user space; CTM-aware tolerance checking is used.
func (r *Rasterizer) flattenQuadratic(p0, p1, p2 vec.Vec2, emit func(from, to vec.Vec2)) {
	// Compute error vector: e = (P0 - 2*P1 + P2) / 4
	e := p0.Sub(p1.Mul(2)).Add(p2).Mul(0.25)

	// Transform to device space
	eDev := r.transformLinear(e)

	// Compute segment count
	n := 1
	errDev := eDev.Length()
	if errDev > r.Flatness {
		// cap before the int conversion so an extreme CTM cannot overflow it
		n = int(min(math.Ceil(math.Sqrt(errDev/r.Flatness)), maxFlattenSegments))
	}

	// Evaluate curve at n+1 points and emit segments
	prev := p0
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		// B(t) = (1-t)²P0 + 2(1-t)tP1 + t²P2
		omt := 1 - t
		pt := p0.Mul(omt * omt).Add(p1.Mul(2 * omt * t)).Add(p2.Mul(t * t))
		emit(prev, pt)
		prev = pt
	}
}

// flattenCubic flattens a cubic Bézier and calls emit for each line segment.
// p0 is start, p1/p2 are controls, p3 is endpoint. All in user space.
func (r *Rasterizer) flattenCubic(p0, p1, p2, p3 vec.Vec2, emit func(from, to vec.Vec2)) {
	// Compute deviation vectors
	d1 := p0.Sub(p1.Mul(2)).Add(p2) // P0 - 2*P1 + P2
	d2 := p1.Sub(p2.Mul(2)).Add(p3) // P1 - 2*P2 + P3

	// Transform to device space
	d1Dev := r.transformLinear(d1)
	d2Dev := r.transformLinear(d2)

	// Compute segment count using Wang's formula
	mDev := max(d1Dev.Length(), d2Dev.Length())
	n := 1
	if mDev > 0 {
		// n = ceil(sqrt(3 * mDev / (4 * ε)))
		nFloat := math.Sqrt(3 * mDev / (4 * r.Flatness))
		if nFloat > 1 {
			// cap before the int conversion so an extreme CTM cannot overflow it
			n = int(min(math.Ceil(nFloat), maxFlattenSegments))
		}
	}

	// Evaluate curve at n+1 points and emit segments
	prev := p0
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		// B(t) = (1-t)³P0 + 3(1-t)²tP1 + 3(1-t)t²P2 + t³P3
		omt := 1 - t
		omt2 := omt * omt
		omt3 := omt2 * omt
		t2 := t * t
		t3 := t2 * t
		pt := p0.Mul(omt3).Add(p1.Mul(3 * omt2 * t)).Add(p2.Mul(3 * omt * t2)).Add(p3.Mul(t3))
		emit(prev, pt)
		prev = pt
	}
}

// FillNonZero fills the path using the nonzero winding rule. The emit
// callback receives coverage row-by-row; its slice argument is valid only
// during the call.
func (r *Rasterizer) FillNonZero(p path.Path, emit func(y, xMin int, coverage []float32)) {
	r.fill(p, fillNonZero, emit)
}

// FillEvenOdd fills the path using the even-odd rule. The emit callback
// receives coverage row-by-row; its slice argument is valid only during
// the call.
func (r *Rasterizer) FillEvenOdd(p path.Path, emit func(y, xMin int, coverage []float32)) {
	r.fill(p, fillEvenOdd, emit)
}

// fillRule identifies which fill rule to apply.
type fillRule int

const (
	fillNonZero fillRule = iota
	fillEvenOdd
)

// fill is the internal implementation shared by FillNonZero and FillEvenOdd.
func (r *Rasterizer) fill(p path.Path, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	if r.Flatness <= 0 {
		panic("raster: Flatness must be positive")
	}

	// Collect edges from path (returns bounding box clamped to clip)
	xMin, xMax, yMin, yMax, ok := r.collectPathEdges(p)
	if !ok {
		return // empty or degenerate path
	}

	// Choose approach based on bounding box size
	width := xMax - xMin
	height := yMax - yMin

	if width*height < r.smallPathThreshold {
		r.fillSmallPath(xMin, xMax, yMin, yMax, rule, emit)
	} else {
		r.fillLargePath(xMin, xMax, yMin, yMax, rule, emit)
	}
}

// collectPathEdges walks the path, transforms to device space, and builds the edge list.
// Returns the bounding box of all edges in device coordinates (clamped to clip).
func (r *Rasterizer) collectPathEdges(p path.Path) (xMin, xMax, yMin, yMax int, ok bool) {
	r.edges = r.edges[:0]
	r.edgeBBoxFirst = true

	// path state
	var current vec.Vec2 // current point (user space)
	var subpath vec.Vec2 // subpath start (user space)
	hasSubpath := false

	for cmd, pts := range p {
		switch cmd {
		case path.CmdMoveTo:
			// implicitly close previous subpath
			if hasSubpath && current != subpath {
				r.addEdge(current, subpath)
			}
			current = pts[0]
			subpath = current
			hasSubpath = true

		case path.CmdLineTo:
			r.addEdge(current, pts[0])
			current = pts[0]

		case path.CmdQuadTo:
			r.flattenQuadratic(current, pts[0], pts[1], r.addEdge)
			current = pts[1]

		case path.CmdCubeTo:
			r.flattenCubic(current, pts[0], pts[1], pts[2], r.addEdge)
			current = pts[2]

		case path.CmdClose:
			if current != subpath {
				r.addEdge(current, subpath)
			}
			current = subpath
			hasSubpath = false
		}
	}

	// implicitly close final subpath
	if hasSubpath && current != subpath {
		r.addEdge(current, subpath)
	}

	if len(r.edges) == 0 {
		return 0, 0, 0, 0, false
	}

	// a non-finite CTM yields NaN extremes; treat the path as degenerate
	if math.IsNaN(r.edgeDevXMin) || math.IsNaN(r.edgeDevXMax) ||
		math.IsNaN(r.edgeDevYMin) || math.IsNaN(r.edgeDevYMax) {
		return 0, 0, 0, 0, false
	}

	// clamp to clip bounds and convert to integers
	clipXMin := int(r.Clip.LLx)
	clipXMax := int(r.Clip.URx)
	clipYMin := int(r.Clip.LLy)
	clipYMax := int(r.Clip.URy)

	// The +1 turns the inclusive max pixel into an exclusive upper bound.
	// Clamping to clipMax-1 first keeps that +1 within the int range even when
	// floorToInt saturates on an extreme CTM.
	xMin = max(floorToInt(r.edgeDevXMin), clipXMin)
	xMax = min(floorToInt(r.edgeDevXMax), clipXMax-1) + 1
	yMin = max(floorToInt(r.edgeDevYMin), clipYMin)
	yMax = min(floorToInt(r.edgeDevYMax), clipYMax-1) + 1

	if xMin >= xMax || yMin >= yMax {
		return 0, 0, 0, 0, false
	}

	return xMin, xMax, yMin, yMax, true
}

// addEdge adds an edge from user space coordinates, transforming to device space.
func (r *Rasterizer) addEdge(p0, p1 vec.Vec2) {
	// Transform to device space
	dx0 := r.CTM[0]*p0.X + r.CTM[2]*p0.Y + r.CTM[4]
	dy0 := r.CTM[1]*p0.X + r.CTM[3]*p0.Y + r.CTM[5]
	dx1 := r.CTM[0]*p1.X + r.CTM[2]*p1.Y + r.CTM[4]
	dy1 := r.CTM[1]*p1.X + r.CTM[3]*p1.Y + r.CTM[5]

	// Skip horizontal edges
	dy := dy1 - dy0
	if dy > -horizontalEdgeThreshold && dy < horizontalEdgeThreshold {
		return
	}

	// Compute dxdy
	dxdy := (dx1 - dx0) / dy

	r.edges = append(r.edges, edge{
		x0: dx0, y0: dy0,
		x1: dx1, y1: dy1,
		dxdy: dxdy,
	})

	// Update bounding box
	if r.edgeBBoxFirst {
		r.edgeDevXMin = min(dx0, dx1)
		r.edgeDevXMax = max(dx0, dx1)
		r.edgeDevYMin = min(dy0, dy1)
		r.edgeDevYMax = max(dy0, dy1)
		r.edgeBBoxFirst = false
	} else {
		r.edgeDevXMin = min(r.edgeDevXMin, min(dx0, dx1))
		r.edgeDevXMax = max(r.edgeDevXMax, max(dx0, dx1))
		r.edgeDevYMin = min(r.edgeDevYMin, min(dy0, dy1))
		r.edgeDevYMax = max(r.edgeDevYMax, max(dy0, dy1))
	}
}

// floorToInt floors f and converts it to int, saturating values outside the
// int range to the int extremes. This keeps the device-space bounding box
// well-defined even when an extreme CTM produces coordinates beyond MaxInt;
// the caller clamps the result to the clip bounds. NaN is not expected here
// (collectPathEdges rejects non-finite extremes first) and maps to zero.
func floorToInt(f float64) int {
	switch {
	case math.IsNaN(f):
		return 0
	case f <= float64(math.MinInt):
		return math.MinInt
	case f >= float64(math.MaxInt):
		return math.MaxInt
	default:
		return int(math.Floor(f))
	}
}

// Coverage accumulation model:
//
// For each pixel, we track two values:
//   cover: signed vertical extent of edges crossing this pixel column
//   area:  horizontal position weighting (how far right the crossing is)
//
// An edge crossing a pixel contributes:
//   cover = sign * dy   (where sign is +1 for downward, -1 for upward)
//   area  = cover * (1 - xFrac)   (where xFrac is the horizontal position within the pixel)
//
// Final coverage is computed by integrateScanline:
//   pixel_coverage = accumulated_cover + area[i]
//   accumulated_cover += cover[i]   (carry forward for next pixel)
//
// This computes the signed area of the path within each pixel, which gives
// anti-aliased coverage values when clamped to [0,1] (nonzero) or folded (even-odd).

// accumulateEdge adds a single edge's contribution to the cover and area buffers.
// The buffers are indexed by (x - bboxXMin), where bboxXMin/bboxXMax define the buffer range.
// For edges spanning multiple pixels horizontally, this function splits the edge at pixel
// boundaries and computes separate contributions for each pixel crossed.
func (r *Rasterizer) accumulateEdge(e *edge, y int, cover, area []float32, bboxXMin, bboxXMax int) {
	// Compute the portion of the edge within this scanline [y, y+1)
	yTop := float64(y)
	yBot := float64(y + 1)

	// Clamp to edge's actual y extent
	edgeYMin := min(e.y0, e.y1)
	edgeYMax := max(e.y0, e.y1)
	yTop = max(yTop, edgeYMin)
	yBot = min(yBot, edgeYMax)

	if yBot <= yTop {
		return
	}

	// Sign based on edge direction: +1 for downward (y1 > y0), -1 for upward
	sign := float32(1)
	if e.y1 < e.y0 {
		sign = -1
	}

	// Compute x at the y boundaries of the edge segment within this scanline
	xAtYTop := e.x0 + e.dxdy*(yTop-e.y0)
	xAtYBot := e.x0 + e.dxdy*(yBot-e.y0)

	// Determine pixel range the edge spans (ensure left <= right for iteration)
	xLeft, xRight := xAtYTop, xAtYBot
	if xLeft > xRight {
		xLeft, xRight = xRight, xLeft
	}

	pixLeft := int(math.Floor(xLeft))
	pixRight := int(math.Floor(xRight))

	// Handle edge entirely to the left of bbox
	if pixRight < bboxXMin {
		coverVal := sign * float32(yBot-yTop)
		cover[0] += coverVal
		area[0] += coverVal
		return
	}

	// Handle edge entirely to the right of bbox
	if pixLeft >= bboxXMax {
		return
	}

	// For vertical edges or edges within a single pixel column
	if pixLeft == pixRight {
		r.accumulateEdgeInColumn(e, yTop, yBot, sign, pixLeft, cover, area, bboxXMin, bboxXMax)
		return
	}

	// Edge spans multiple pixels - process each pixel column in x-order
	// For each pixel, compute the y-extent of the edge within that column
	dydx := 1 / e.dxdy

	for pix := pixLeft; pix <= pixRight; pix++ {
		// Compute y at column boundaries
		yAtPixLeft := e.y0 + dydx*(float64(pix)-e.x0)
		yAtPixRight := e.y0 + dydx*(float64(pix+1)-e.x0)

		// Clamp to edge's y-extent within scanline
		segYMin := max(min(yAtPixLeft, yAtPixRight), yTop)
		segYMax := min(max(yAtPixLeft, yAtPixRight), yBot)

		segDy := segYMax - segYMin
		if segDy <= 0 {
			continue
		}

		// Compute contribution for this segment
		coverVal := sign * float32(segDy)

		// Compute average x within this pixel column
		yMid := (segYMin + segYMax) / 2
		xMid := e.x0 + e.dxdy*(yMid-e.y0)
		xFrac := xMid - float64(pix)
		areaVal := coverVal * float32(1-xFrac)

		// Add to buffers
		if pix < bboxXMin {
			cover[0] += coverVal
			area[0] += coverVal
		} else if pix < bboxXMax {
			idx := pix - bboxXMin
			cover[idx] += coverVal
			area[idx] += areaVal
		}
		// pix >= bboxXMax: no contribution
	}
}

// accumulateEdgeInColumn handles an edge segment that falls within a single pixel column.
func (r *Rasterizer) accumulateEdgeInColumn(e *edge, yTop, yBot float64, sign float32, pix int, cover, area []float32, bboxXMin, bboxXMax int) {
	coverVal := sign * float32(yBot-yTop)

	if pix < bboxXMin {
		cover[0] += coverVal
		area[0] += coverVal
		return
	}
	if pix >= bboxXMax {
		return
	}

	// Compute average x within this pixel
	yMid := (yTop + yBot) / 2
	xMid := e.x0 + e.dxdy*(yMid-e.y0)
	xFrac := xMid - float64(pix)
	areaVal := coverVal * float32(1-xFrac)

	idx := pix - bboxXMin
	cover[idx] += coverVal
	area[idx] += areaVal
}

// integrateScanlineNonZero converts accumulated cover/area to final coverage
// values using the nonzero winding rule. The cover slice is modified in place.
func integrateScanlineNonZero(cover, area []float32) {
	area = area[:len(cover)] // bounds check elimination hint
	var accum float32
	for i := range cover {
		raw := accum + area[i]
		accum += cover[i]

		// clamp(abs(raw), 0, 1)
		cov := raw
		if raw < 0 {
			cov = -raw
		}
		if cov > 1 {
			cov = 1
		}
		cover[i] = cov
	}
}

// integrateScanlineEvenOdd converts accumulated cover/area to final coverage
// values using the even-odd fill rule. The cover slice is modified in place.
func integrateScanlineEvenOdd(cover, area []float32) {
	area = area[:len(cover)] // bounds check elimination hint
	var accum float32
	for i := range cover {
		raw := accum + area[i]
		accum += cover[i]

		// even-odd: 1 - |1 - mod(|raw|, 2)|
		// for |raw| ≤ 1 this simplifies to |raw|
		if raw < 0 {
			raw = -raw
		}
		if raw <= 1 {
			cover[i] = raw
			continue
		}
		mod := raw - 2*float32(int(raw/2))
		cover[i] = 1 - abs32(1-mod)
	}
}

// abs32 returns the absolute value of a float32.
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// trimZeros returns the non-zero portion of coverage and its starting offset.
// Returns nil, 0 if coverage is entirely zero.
func trimZeros(coverage []float32) (trimmed []float32, offset int) {
	n := len(coverage)
	lo := 0
	for lo < n && coverage[lo] == 0 {
		lo++
	}
	if lo == n {
		return nil, 0
	}
	hi := n - 1
	for hi > lo && coverage[hi] == 0 {
		hi--
	}
	return coverage[lo : hi+1], lo
}

// fillSmallPath rasterises using 2D buffers (Approach A).
// Used for small paths where width*height < smallPathThreshold.
// xMin, xMax, yMin, yMax define the path's bounding box (already clamped to clip).
func (r *Rasterizer) fillSmallPath(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	width := xMax - xMin
	height := yMax - yMin

	// Ensure 2D buffers are large enough and zero them
	size := width * height
	r.cover = slices.Grow(r.cover[:0], size)[:size]
	r.area = slices.Grow(r.area[:0], size)[:size]
	clear(r.cover)
	clear(r.area)

	// Ensure row tracking buffer is large enough and clear it
	r.rowHasEdges = slices.Grow(r.rowHasEdges[:0], height)[:height]
	clear(r.rowHasEdges)

	// Process all edges into 2D buffers
	for i := range r.edges {
		e := &r.edges[i]

		// Determine scanline range for this edge
		var edgeYMin, edgeYMax int
		if e.y0 < e.y1 {
			edgeYMin = int(math.Floor(e.y0))
			edgeYMax = int(math.Floor(e.y1)) + 1
		} else {
			edgeYMin = int(math.Floor(e.y1))
			edgeYMax = int(math.Floor(e.y0)) + 1
		}
		edgeYMin = max(edgeYMin, yMin)
		edgeYMax = min(edgeYMax, yMax)

		// Accumulate into each scanline
		for y := edgeYMin; y < edgeYMax; y++ {
			row := y - yMin
			rowOffset := row * width
			r.accumulateEdge(e, y, r.cover[rowOffset:rowOffset+width], r.area[rowOffset:rowOffset+width], xMin, xMax)
			r.rowHasEdges[row] = true
		}
	}

	// Integrate and emit each row
	for row := range height {
		if !r.rowHasEdges[row] {
			continue // no edges touched this row
		}

		y := yMin + row
		rowOffset := row * width

		// Integrate the full width (cover accumulates from left)
		coverage := r.cover[rowOffset : rowOffset+width]
		if rule == fillNonZero {
			integrateScanlineNonZero(coverage, r.area[rowOffset:rowOffset+width])
		} else {
			integrateScanlineEvenOdd(coverage, r.area[rowOffset:rowOffset+width])
		}

		// Emit only the non-zero portion
		if trimmed, offset := trimZeros(coverage); trimmed != nil {
			emit(y, xMin+offset, trimmed)
		}
	}
}

// fillLargePath rasterises using 1D buffers and an active edge list (Approach B).
// Used for large paths where width*height >= smallPathThreshold.
// xMin, xMax, yMin, yMax define the path's bounding box (already clamped to clip).
func (r *Rasterizer) fillLargePath(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	width := xMax - xMin

	// Ensure 1D buffers are large enough
	r.cover = slices.Grow(r.cover[:0], width)[:width]
	r.area = slices.Grow(r.area[:0], width)[:width]

	// Sort edges by y_min
	slices.SortFunc(r.edges, func(a, b edge) int {
		aYMin := min(a.y0, a.y1)
		bYMin := min(b.y0, b.y1)
		return cmp.Compare(aYMin, bYMin)
	})

	// Active edge list (indices into r.edges)
	r.activeIdx = r.activeIdx[:0]
	nextEdge := 0

	// Process scanlines
	for y := yMin; y < yMax; y++ {
		yf := float64(y)
		yfNext := float64(y + 1)

		// Add edges that start at this scanline
		for nextEdge < len(r.edges) {
			e := &r.edges[nextEdge]
			edgeYMin := min(e.y0, e.y1)
			if edgeYMin >= yfNext {
				break
			}
			r.activeIdx = append(r.activeIdx, nextEdge)
			nextEdge++
		}

		if len(r.activeIdx) == 0 {
			continue
		}

		// Clear buffers for this scanline
		clear(r.cover)
		clear(r.area)

		// Track x bounds for this scanline
		xMinBound := width
		xMaxBound := -1

		// Process active edges
		for i := 0; i < len(r.activeIdx); {
			e := &r.edges[r.activeIdx[i]]

			// Check if edge ends before this scanline
			edgeYMax := max(e.y0, e.y1)
			if edgeYMax <= yf {
				// Remove from active list (swap with last)
				r.activeIdx[i] = r.activeIdx[len(r.activeIdx)-1]
				r.activeIdx = r.activeIdx[:len(r.activeIdx)-1]
				continue
			}

			// Accumulate contribution
			r.accumulateEdge(e, y, r.cover, r.area, xMin, xMax)

			// Update x bounds
			yTop := max(yf, min(e.y0, e.y1))
			yBot := min(yfNext, max(e.y0, e.y1))
			if yBot > yTop {
				yMid := (yTop + yBot) / 2
				xMidF := e.x0 + e.dxdy*(yMid-e.y0)
				x := int(math.Floor(xMidF))
				x = max(x, xMin)
				x = min(x, xMax-1)
				xIdx := x - xMin
				if xIdx < xMinBound {
					xMinBound = xIdx
				}
				if xIdx > xMaxBound {
					xMaxBound = xIdx
				}
			}

			i++
		}

		if xMaxBound < 0 {
			continue // no edges contributed to this scanline
		}

		// Integrate and emit
		if rule == fillNonZero {
			integrateScanlineNonZero(r.cover, r.area)
		} else {
			integrateScanlineEvenOdd(r.cover, r.area)
		}

		// Emit only the non-zero portion
		if trimmed, offset := trimZeros(r.cover); trimmed != nil {
			emit(y, xMin+offset, trimmed)
		}
	}
}

// Default values for rasterizer parameters.
const (
	// defaultFlatness is the default curve flattening tolerance in device
	// pixels. Values of 0.25-1.0 are typical; 0.25 is below the threshold
	// of visual perception.
	defaultFlatness = 0.25

	// defaultMiterLimit is the default miter limit, matching PDF/PostScript.
	// This converts joins to bevels when the interior angle is less than
	// approximately 11.5 degrees.
	defaultMiterLimit = 10.0
)

// Numerical tolerances for the rasterizer.
const (
	// horizontalEdgeThreshold is the minimum vertical extent for an edge
	// to contribute to coverage. Edges with |y1 - y0| below this threshold
	// are skipped as horizontal.
	horizontalEdgeThreshold = 1e-10

	// smallPathThreshold is the maximum bounding box area (in pixels) for
	// using 2D buffers (Approach A). Paths with larger bounding boxes use
	// the active edge list (Approach B).
	// TODO: tune this threshold based on profiling
	smallPathThreshold = 65536

	// zeroLengthThreshold is the minimum length for a stroke segment.
	// Segments shorter than this are skipped.
	zeroLengthThreshold = 1e-10

	// collinearityThreshold is used to detect nearly collinear segments
	// where no join is needed.
	collinearityThreshold = 1e-6

	// cuspCosineThreshold is the cosine threshold for detecting cusps
	// (path doubling back on itself). cos(179.43°) ≈ -0.9999
	cuspCosineThreshold = -0.9999

	// maxFlattenSegments bounds the number of line segments a single curve or
	// arc is flattened into. A curve never needs more segments than there are
	// device pixels along it, so this ceiling cannot affect legitimate output
	// (the segment count stays in the low hundreds even on a large canvas).
	// It stops an attacker-controlled CTM scale from driving the count out of
	// range and hanging the rasterizer.
	maxFlattenSegments = 1 << 16

	// maxDashSegments bounds the number of dash pieces a single subpath is
	// split into. A pattern fine enough to exceed this is sub-pixel on any
	// reasonable canvas (the piece count is independent of the CTM), so the
	// ceiling cannot affect legitimate output; it stops an attacker-controlled
	// dash array from driving the dash loop out of range and hanging the
	// rasterizer.
	maxDashSegments = 1 << 16
)
