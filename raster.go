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
	"image"
	"math"
	"slices"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// Rasterizer converts vector paths to pixel coverage fractions. Coverage
// fractions are represented by float32 values in the range from 0 (outside) to
// 1 (inside).
//
// All exported fields are user-settable graphics parameters. Drawing
// operations are adjusted by setting these fields before calling one of the
// drawing methods.
//
// A Rasterizer is not safe for concurrent use.
type Rasterizer struct {
	// CTM transforms from user space to device space. Must be non-singular.
	CTM matrix.Matrix

	// Clip bounds output to this half-open rectangle of device pixels.
	// If the rectangle is empty or not well-formed it produces no output.
	Clip image.Rectangle

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
	bucketHead    []int32    // per-scanline chain heads (edge index or -1)
	bucketNext    []int32    // next edge in the same scanline bucket
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
	dashedClosed      []bool          // whether each dashed subpath is closed

	// ctmSigmaMax is the largest singular value of CTM's linear part,
	// computed once per Stroke call and used to bound device-space scaling
	// for join folding and arc segment counts.
	ctmSigmaMax float64
}

// NewRasterizer returns a Rasterizer with the given clip rectangle and
// PDF default values for other parameters.
func NewRasterizer(clip image.Rectangle) *Rasterizer {
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
	if r.Clip.Empty() {
		return // skip flattening and edge collection when nothing can be emitted
	}

	// Collect edges from path (returns bounding box clamped to clip)
	xMin, xMax, yMin, yMax, ok := r.collectPathEdges(p)
	if !ok {
		return // empty or degenerate path
	}

	r.rasterizeEdges(xMin, xMax, yMin, yMax, rule, emit)
}

// rasterizeEdges rasterises the edges collected in r.edges within the given
// bounding box, choosing the small-path (2D buffer) or large-path (active
// edge list) approach by its pixel area.
func (r *Rasterizer) rasterizeEdges(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
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

	return r.clippedEdgeBounds()
}

// clippedEdgeBounds converts the device-space bounding box accumulated by
// addEdge into a half-open pixel rectangle clamped to Clip.  ok is false if
// nothing is left to rasterise.
func (r *Rasterizer) clippedEdgeBounds() (xMin, xMax, yMin, yMax int, ok bool) {
	// An empty clip admits no pixels.  Returning early also keeps the Max-1
	// below from underflowing, since a non-empty rectangle has Max > Min.
	if r.Clip.Empty() {
		return 0, 0, 0, 0, false
	}

	// The +1 turns the inclusive max pixel into an exclusive upper bound.
	// Clamping to Max-1 first keeps that +1 within the int range even when
	// floorToInt saturates on an extreme CTM.
	xMin = max(floorToInt(r.edgeDevXMin), r.Clip.Min.X)
	xMax = min(floorToInt(r.edgeDevXMax), r.Clip.Max.X-1) + 1
	yMin = max(floorToInt(r.edgeDevYMin), r.Clip.Min.Y)
	yMax = min(floorToInt(r.edgeDevYMax), r.Clip.Max.Y-1) + 1

	if xMin >= xMax || yMin >= yMax {
		return 0, 0, 0, 0, false
	}

	return xMin, xMax, yMin, yMax, true
}

// nonZeroCoverage converts a raw winding value to coverage under the
// nonzero winding rule.
func nonZeroCoverage(raw float32) float32 {
	if raw < 0 {
		raw = -raw
	}
	return min(raw, 1)
}

// evenOddCoverage converts a raw winding value to coverage under the
// even-odd rule.
func evenOddCoverage(raw float32) float32 {
	// 1 - |1 - mod(|raw|, 2)|, which for |raw| ≤ 1 is just |raw|
	if raw < 0 {
		raw = -raw
	}
	if raw <= 1 {
		return raw
	}
	mod := raw - 2*float32(int(raw/2))
	return 1 - abs32(1-mod)
}

// integrateScanlineNonZero converts accumulated cover/area to final coverage
// values using the nonzero winding rule. The cover slice is modified in
// place.  The return value is the winding carried past the end of the slice.
func integrateScanlineNonZero(cover, area []float32) float32 {
	area = area[:len(cover)] // bounds check elimination hint
	var accum float32
	for i := range cover {
		raw := accum + area[i]
		accum += cover[i]
		cover[i] = nonZeroCoverage(raw)
	}
	return accum
}

// integrateScanlineEvenOdd converts accumulated cover/area to final coverage
// values using the even-odd fill rule. The cover slice is modified in place.
// The return value is the winding carried past the end of the slice.
func integrateScanlineEvenOdd(cover, area []float32) float32 {
	area = area[:len(cover)] // bounds check elimination hint
	var accum float32
	for i := range cover {
		raw := accum + area[i]
		accum += cover[i]
		cover[i] = evenOddCoverage(raw)
	}
	return accum
}

// floorToInt floors f and converts it to int, saturating values outside the
// int range to the int extremes and mapping NaN to zero.  Edges are clipped
// to the clip rectangle, so saturation only matters for clips near the int
// extremes.  NaN should not reach here, but Go leaves int(math.Floor(NaN))
// implementation-defined -- it is 0 on arm64 and MinInt on amd64 -- so the
// result is pinned to keep rendering architecture-independent.
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

		// Determine scanline range for this edge.  Clamping before the +1
		// keeps it in range, as in clippedEdgeBounds.
		edgeYMin := max(floorToInt(e.ymin), yMin)
		edgeYMax := min(floorToInt(e.ymax), yMax-1) + 1

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

	// Ensure 1D buffers are large enough and start out zero.  Each row
	// clears only the part it wrote, so the buffers stay zero between rows.
	r.cover = slices.Grow(r.cover[:0], width)[:width]
	r.area = slices.Grow(r.area[:0], width)[:width]
	clear(r.cover)
	clear(r.area)

	// Bucket edges by starting scanline. Order within a scanline does not
	// matter: contributions are additive.
	height := yMax - yMin
	r.bucketHead = slices.Grow(r.bucketHead[:0], height)[:height]
	for i := range r.bucketHead {
		r.bucketHead[i] = -1
	}
	r.bucketNext = slices.Grow(r.bucketNext[:0], len(r.edges))[:len(r.edges)]
	for i := range r.edges {
		b := floorToInt(r.edges[i].ymin) - yMin
		b = max(b, 0)
		b = min(b, height-1)
		r.bucketNext[i] = r.bucketHead[b]
		r.bucketHead[b] = int32(i)
	}

	// Active edge list (indices into r.edges)
	r.activeIdx = r.activeIdx[:0]

	// Process scanlines
	for y := yMin; y < yMax; y++ {
		yf := float64(y)

		// Add edges that start at this scanline
		for idx := r.bucketHead[y-yMin]; idx >= 0; idx = r.bucketNext[idx] {
			r.activeIdx = append(r.activeIdx, int(idx))
		}

		if len(r.activeIdx) == 0 {
			continue
		}

		// Process active edges, tracking the span of buffer cells written.
		lo, hi := width, -1
		for i := 0; i < len(r.activeIdx); {
			e := &r.edges[r.activeIdx[i]]

			// Check if edge ends before this scanline
			if e.ymax <= yf {
				// Remove from active list (swap with last)
				r.activeIdx[i] = r.activeIdx[len(r.activeIdx)-1]
				r.activeIdx = r.activeIdx[:len(r.activeIdx)-1]
				continue
			}

			eLo, eHi := r.accumulateEdge(e, y, r.cover, r.area, xMin, xMax)
			if eLo <= eHi {
				lo = min(lo, eLo)
				hi = max(hi, eHi)
			}

			i++
		}

		if hi < 0 {
			continue // no edges contributed to this scanline
		}

		// Integrate the written span.  Winding carried past its end covers
		// the rest of the row uniformly; a residual below coverEpsilon is
		// float error from cancelling edges and counts as zero.
		span := r.cover[lo : hi+1]
		var carry float32
		if rule == fillNonZero {
			carry = integrateScanlineNonZero(span, r.area[lo:hi+1])
		} else {
			carry = integrateScanlineEvenOdd(span, r.area[lo:hi+1])
		}
		end := hi + 1
		if carry > coverEpsilon || carry < -coverEpsilon {
			var tail float32
			if rule == fillNonZero {
				tail = nonZeroCoverage(carry)
			} else {
				tail = evenOddCoverage(carry)
			}
			end = width
			for i := hi + 1; i < end; i++ {
				r.cover[i] = tail
			}
		}

		// Emit only the non-zero portion
		if trimmed, offset := trimZeros(r.cover[lo:end]); trimmed != nil {
			emit(y, xMin+lo+offset, trimmed)
		}

		// restore the zero state for the next row
		clear(r.cover[lo:end])
		clear(r.area[lo : hi+1])
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

	// coverEpsilon is the winding residual below which a row's carried-over
	// winding counts as zero.  The covers of the edges crossing a row cancel
	// exactly for a closed path, up to float32 rounding.
	coverEpsilon = 1e-6

	// collinearityThreshold is the turn angle below which segments count as
	// collinear and no join is needed.
	collinearityThreshold = 1e-6

	// maxStripTurn bounds the accumulated turn angle within a single strip of
	// the stroke outline. Below half a turn all tangents of a strip fit in a
	// quarter-turn cone, which keeps both offset boundaries advancing in one
	// common direction and hence the strip simple.
	maxStripTurn = math.Pi / 2

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
