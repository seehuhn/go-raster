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
	"math"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// strokeSegment represents a line segment in user coordinates
type strokeSegment struct {
	A, B vec.Vec2 // endpoints in user space
	T    vec.Vec2 // unit tangent (A→B direction)
	N    vec.Vec2 // unit normal (90° CCW from T)
}

// Stroke renders the path as a stroked outline using Width, Cap, Join,
// MiterLimit, Dash, and DashPhase. The emit callback receives coverage
// row-by-row; its slice argument is valid only during the call.
func (r *Rasterizer) Stroke(p path.Path, emit func(y, xMin int, coverage []float32)) {
	if r.Flatness <= 0 {
		panic("raster: Flatness must be positive")
	}
	if r.Width <= 0 {
		panic("raster: Width must be positive")
	}

	// Flatten path into subpaths (results stored in r.segs, etc.)
	r.flattenPath(p)
	if len(r.segsOffsets) == 0 && len(r.degeneratePoints) == 0 {
		return
	}

	// Build stroke outlines for all subpaths into a single contiguous buffer.
	// strokeOffsets tracks where each polygon starts. This ensures overlapping
	// dash segments are composited correctly using the nonzero winding rule.
	r.stroke = r.stroke[:0]
	r.strokeOffsets = r.strokeOffsets[:0]

	// Handle degenerate subpaths (no orientation): only round cap produces circle
	if r.Cap == graphics.LineCapRound {
		for _, pt := range r.degeneratePoints {
			startOffset := len(r.stroke)
			r.addArc(pt, r.Width/2, vec.Vec2{X: 1, Y: 0}, 2*math.Pi, true)
			r.strokeOffsets = append(r.strokeOffsets, startOffset)
		}
	}

	// Apply dash pattern if specified
	if len(r.Dash) > 0 {
		r.strokeDashedSubpaths()
	} else {
		r.strokeAllSubpaths()
	}

	// Fill all stroke polygons together as a compound path
	r.fillStrokeOutlines(emit)
}

// strokeAllSubpaths strokes all flattened subpaths (non-dashed case).
func (r *Rasterizer) strokeAllSubpaths() {
	numSubpaths := len(r.segsOffsets)
	for i := range numSubpaths {
		segs := r.getSubpathSegments(i)
		closed := r.subpathClosed[i]

		startOffset := len(r.stroke)
		r.strokeSubpath(segs, closed)
		if len(r.stroke)-startOffset >= 3 {
			r.strokeOffsets = append(r.strokeOffsets, startOffset)
		} else {
			// Degenerate polygon, discard by resetting to start
			r.stroke = r.stroke[:startOffset]
		}
	}
}

// getSubpathSegments returns the segments for subpath i as a slice into segs.
func (r *Rasterizer) getSubpathSegments(i int) []strokeSegment {
	start := r.segsOffsets[i]
	var end int
	if i+1 < len(r.segsOffsets) {
		end = r.segsOffsets[i+1]
	} else {
		end = len(r.segs)
	}
	return r.segs[start:end]
}

// strokeDashedSubpaths applies dash pattern and strokes the resulting segments.
func (r *Rasterizer) strokeDashedSubpaths() {
	// Apply dash pattern - populates r.dashedSegs and r.dashedSegsOffsets
	r.applyDashPattern()

	numDashes := len(r.dashedSegsOffsets)
	for i := range numDashes {
		segs := r.getDashedSegments(i)

		// Handle dash-created zero-length segments (have orientation from underlying path)
		if len(segs) == 1 && segs[0].A == segs[0].B {
			seg := &segs[0]
			startOffset := len(r.stroke)
			switch r.Cap {
			case graphics.LineCapRound:
				r.addArc(seg.A, r.Width/2, vec.Vec2{X: 1, Y: 0}, 2*math.Pi, true)
				r.strokeOffsets = append(r.strokeOffsets, startOffset)
			case graphics.LineCapSquare:
				r.addSquare(seg.A, seg.T, r.Width/2)
				r.strokeOffsets = append(r.strokeOffsets, startOffset)
			}
			// Butt cap: no output
			continue
		}

		startOffset := len(r.stroke)
		r.strokeSubpath(segs, false) // dashed subpaths are never closed
		if len(r.stroke)-startOffset >= 3 {
			r.strokeOffsets = append(r.strokeOffsets, startOffset)
		} else {
			// Degenerate polygon, discard by resetting to start
			r.stroke = r.stroke[:startOffset]
		}
	}
}

// getDashedSegments returns the segments for dashed subpath i as a slice into dashedSegs.
func (r *Rasterizer) getDashedSegments(i int) []strokeSegment {
	start := r.dashedSegsOffsets[i]
	var end int
	if i+1 < len(r.dashedSegsOffsets) {
		end = r.dashedSegsOffsets[i+1]
	} else {
		end = len(r.dashedSegs)
	}
	return r.dashedSegs[start:end]
}

// flattenPath walks the path, flattens curves, and populates the flattening
// buffers with precomputed segment geometry. Results are stored in:
//   - r.segs: all segments from all subpaths, contiguous
//   - r.segsOffsets: start index of each subpath in segs
//   - r.subpathClosed: whether each subpath is closed
//   - r.degeneratePoints: degenerate subpaths (no orientation)
func (r *Rasterizer) flattenPath(p path.Path) {
	// clear buffers (preserving capacity)
	r.segs = r.segs[:0]
	r.segsOffsets = r.segsOffsets[:0]
	r.subpathClosed = r.subpathClosed[:0]
	r.degeneratePoints = r.degeneratePoints[:0]

	var currentPt vec.Vec2
	var subpathStartPt vec.Vec2
	subpathStartIdx := 0 // index into flattenedSegs where current subpath starts
	inSubpath := false
	sawDrawingCmd := false // tracks if we saw LineTo/QuadTo/CubeTo (for degenerate detection)

	for cmd, pts := range p {
		switch cmd {
		case path.CmdMoveTo:
			// close previous subpath if needed
			if inSubpath && (len(r.segs) > subpathStartIdx || sawDrawingCmd) {
				if len(r.segs) == subpathStartIdx {
					// degenerate subpath (no orientation) - collect for special handling
					r.degeneratePoints = append(r.degeneratePoints, subpathStartPt)
				} else {
					r.segsOffsets = append(r.segsOffsets, subpathStartIdx)
					r.subpathClosed = append(r.subpathClosed, false)
				}
			}
			currentPt = pts[0]
			subpathStartPt = currentPt
			subpathStartIdx = len(r.segs)
			inSubpath = true
			sawDrawingCmd = false

		case path.CmdLineTo:
			if !inSubpath {
				continue
			}
			sawDrawingCmd = true
			r.addStrokeSegment(currentPt, pts[0])
			currentPt = pts[0]

		case path.CmdQuadTo:
			if !inSubpath {
				continue
			}
			sawDrawingCmd = true
			r.flattenQuadratic(currentPt, pts[0], pts[1], r.addStrokeSegment)
			currentPt = pts[1]

		case path.CmdCubeTo:
			if !inSubpath {
				continue
			}
			sawDrawingCmd = true
			r.flattenCubic(currentPt, pts[0], pts[1], pts[2], r.addStrokeSegment)
			currentPt = pts[2]

		case path.CmdClose:
			if inSubpath {
				// add closing segment if needed
				if currentPt != subpathStartPt {
					r.addStrokeSegment(currentPt, subpathStartPt)
				}
				if len(r.segs) == subpathStartIdx {
					// degenerate closed subpath - collect for special handling
					r.degeneratePoints = append(r.degeneratePoints, subpathStartPt)
				} else {
					r.segsOffsets = append(r.segsOffsets, subpathStartIdx)
					r.subpathClosed = append(r.subpathClosed, true)
				}
				currentPt = subpathStartPt
				subpathStartIdx = len(r.segs)
				inSubpath = false
				sawDrawingCmd = false
			}
		}
	}

	// handle unclosed subpath at end
	if inSubpath && (len(r.segs) > subpathStartIdx || sawDrawingCmd) {
		if len(r.segs) == subpathStartIdx {
			// degenerate subpath - collect for special handling
			r.degeneratePoints = append(r.degeneratePoints, subpathStartPt)
		} else {
			r.segsOffsets = append(r.segsOffsets, subpathStartIdx)
			r.subpathClosed = append(r.subpathClosed, false)
		}
	}
}

// addStrokeSegment adds a line segment to the flattening buffer.
func (r *Rasterizer) addStrokeSegment(a, b vec.Vec2) {
	d := b.Sub(a)
	length := d.Length()
	if length < zeroLengthThreshold {
		return // skip degenerate segment
	}
	t := d.Mul(1 / length)         // unit tangent
	n := vec.Vec2{X: -t.Y, Y: t.X} // unit normal (90° CCW)
	r.segs = append(r.segs, strokeSegment{A: a, B: b, T: t, N: n})
}

// strokeSubpath builds the stroke outline for a single subpath into r.stroke.
// The stroke outline is built as a closed polygon: forward pass on the +N side,
// then backward pass on the -N side. Join geometry is added on the outer side
// of each corner, which depends on the turn direction.
// Zero-length subpaths are handled by the caller before invoking this method.
func (r *Rasterizer) strokeSubpath(segs []strokeSegment, closed bool) {
	if len(segs) == 0 {
		return // empty, nothing to do
	}

	d := r.Width / 2 // half-width

	if closed {
		// Closed path: no caps, just joins
		// Build one continuous polygon: +N side forward, then -N side backward
		// The closing corner needs special handling to connect the two sides.

		first := &segs[0]
		last := &segs[len(segs)-1]

		// Forward pass: +N side (right side of path direction)
		// Start with the closing corner's +N point from segment 0's perspective
		sinThetaClose := last.T.X*first.T.Y - last.T.Y*first.T.X
		r.stroke = append(r.stroke, first.A.Add(first.N.Mul(d)))
		for i := range len(segs) {
			seg := &segs[i]
			if i < len(segs)-1 {
				next := &segs[i+1]
				sinTheta := seg.T.X*next.T.Y - seg.T.Y*next.T.X
				if math.Abs(sinTheta) < collinearityThreshold {
					// Nearly collinear: just add offset points
					r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
					r.stroke = append(r.stroke, next.A.Add(next.N.Mul(d)))
				} else if sinTheta > 0 {
					// Right turn: +N is inner side
					r.addInnerIntersectionOrOffsets(seg.B, seg.T, next.T, seg.N, next.N, d, true)
				} else {
					// Left turn: +N is outer side
					r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
					r.addJoin(seg.B, seg.T, next.T, d, true)
					r.stroke = append(r.stroke, next.A.Add(next.N.Mul(d)))
				}
			} else {
				// Last segment: handle closing corner
				if math.Abs(sinThetaClose) < collinearityThreshold {
					// Nearly collinear: add both offset points
					r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
					r.stroke = append(r.stroke, first.A.Add(first.N.Mul(d)))
				} else if sinThetaClose > 0 {
					// Right turn: +N is inner side - intersection replaces seg.B and first.A
					r.addInnerIntersectionOrOffsets(seg.B, seg.T, first.T, seg.N, first.N, d, true)
				} else {
					// Left turn: +N is outer side
					r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
					r.addJoin(seg.B, seg.T, first.T, d, true)
					r.stroke = append(r.stroke, first.A.Add(first.N.Mul(d)))
				}
			}
		}

		// Backward pass: -N side (left side of path direction)
		// Handle closing corner first, then iterate backwards through segments
		if math.Abs(sinThetaClose) < collinearityThreshold {
			// Nearly collinear: add both offset points
			r.stroke = append(r.stroke, first.A.Sub(first.N.Mul(d)))
			r.stroke = append(r.stroke, last.B.Sub(last.N.Mul(d)))
		} else if sinThetaClose > 0 {
			// Right turn: -N is outer side
			r.stroke = append(r.stroke, first.A.Sub(first.N.Mul(d)))
			r.addJoin(first.A, last.T, first.T, d, false)
			r.stroke = append(r.stroke, last.B.Sub(last.N.Mul(d)))
		} else {
			// Left turn: -N is inner side - intersection replaces first.A and last.B
			r.addInnerIntersectionOrOffsets(first.A, last.T, first.T, last.N, first.N, d, false)
		}

		for i := len(segs) - 1; i >= 0; i-- {
			seg := &segs[i]
			// Add join at this segment's A point (corner with previous segment)
			if i > 0 {
				prev := &segs[i-1]
				sinTheta := prev.T.X*seg.T.Y - prev.T.Y*seg.T.X
				if math.Abs(sinTheta) < collinearityThreshold {
					// Nearly collinear: just add offset points
					r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
					r.stroke = append(r.stroke, prev.B.Sub(prev.N.Mul(d)))
				} else if sinTheta > 0 {
					// Right turn: -N is outer side
					r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
					r.addJoin(seg.A, prev.T, seg.T, d, false)
					r.stroke = append(r.stroke, prev.B.Sub(prev.N.Mul(d)))
				} else {
					// Left turn: -N is inner side
					r.addInnerIntersectionOrOffsets(seg.A, prev.T, seg.T, prev.N, seg.N, d, false)
				}
			} else {
				// First segment (i=0): add closing point of polygon
				r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
			}
		}

	} else {
		// Open path: caps at ends, joins in between
		first := &segs[0]
		last := &segs[len(segs)-1]

		// Start cap (at first.A, direction = -T)
		r.addCap(first.A, first.T.Mul(-1), d)

		// Forward pass: +N side (right side of path direction)
		skipNextA := false
		for i := range len(segs) {
			seg := &segs[i]
			if !skipNextA {
				r.stroke = append(r.stroke, seg.A.Add(seg.N.Mul(d)))
			}
			skipNextA = false
			if i < len(segs)-1 {
				next := &segs[i+1]
				sinTheta := seg.T.X*next.T.Y - seg.T.Y*next.T.X
				if math.Abs(sinTheta) < collinearityThreshold {
					// Nearly collinear: just add offset points
					r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
				} else if sinTheta > 0 {
					// Right turn: +N is inner side
					skipNextA = r.addInnerIntersectionOrOffsets(seg.B, seg.T, next.T, seg.N, next.N, d, true)
				} else {
					// Left turn: +N is outer side
					r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
					r.addJoin(seg.B, seg.T, next.T, d, true)
				}
			} else {
				r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
			}
		}

		// End cap (at last.B, direction = T)
		r.addCap(last.B, last.T, d)

		// Backward pass: -N side (left side of path direction)
		skipNextB := false
		for i := len(segs) - 1; i >= 0; i-- {
			seg := &segs[i]
			if !skipNextB {
				r.stroke = append(r.stroke, seg.B.Sub(seg.N.Mul(d)))
			}
			skipNextB = false
			// Add join at this segment's A point (corner with previous segment)
			if i > 0 {
				prev := &segs[i-1]
				sinTheta := prev.T.X*seg.T.Y - prev.T.Y*seg.T.X
				if math.Abs(sinTheta) < collinearityThreshold {
					// Nearly collinear: just add offset points
					r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
				} else if sinTheta > 0 {
					// Right turn: -N is outer side
					r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
					r.addJoin(seg.A, prev.T, seg.T, d, false)
				} else {
					// Left turn: -N is inner side
					skipNextB = r.addInnerIntersectionOrOffsets(seg.A, prev.T, seg.T, prev.N, seg.N, d, false)
				}
			} else {
				r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
			}
		}
	}
}

// addCap adds a line cap to the stroke outline at point P.
// T is the outward tangent direction (away from the line).
// d is half the stroke width.
func (r *Rasterizer) addCap(P, T vec.Vec2, d float64) {
	N := vec.Vec2{X: -T.Y, Y: T.X} // normal (90° CCW from T)

	switch r.Cap {
	case graphics.LineCapButt:
		// Butt cap: just connect left and right offset points (already done by caller)
		// No additional points needed

	case graphics.LineCapSquare:
		// Square cap: extend by d along tangent
		ext := P.Add(T.Mul(d))
		left := ext.Add(N.Mul(d))
		right := ext.Sub(N.Mul(d))
		r.stroke = append(r.stroke, left, right)

	case graphics.LineCapRound:
		// Round cap: semicircular arc curving outward (through T direction)
		// Arc starts at N direction and sweeps CW (negative angle) to reach -N,
		// passing through T (the outward direction)
		// includeStart=true because cap's start point is not yet in the polygon
		r.addArc(P, d, N, -math.Pi, true)
	}
}

// computeInnerIntersection returns the intersection point of the two inner
// offset lines at a corner. Returns the point and ok=true if valid.
// For nearly collinear segments, returns ok=false.
func computeInnerIntersection(P, T1, T2 vec.Vec2, d float64, isPositiveNormalSide bool) (vec.Vec2, bool) {
	cosTheta := T1.Dot(T2)

	// Nearly collinear - no meaningful intersection
	if cosTheta > 1-1e-9 {
		return vec.Vec2{}, false
	}

	// half_angle = cos(θ/2) = sqrt((1 + cos_θ) / 2)
	halfAngle := math.Sqrt((1 + cosTheta) / 2)
	if halfAngle < 1e-9 {
		return vec.Vec2{}, false
	}

	N1 := vec.Vec2{X: -T1.Y, Y: T1.X}
	N2 := vec.Vec2{X: -T2.Y, Y: T2.X}

	// Inner direction: for +N inner, use N1+N2; for -N inner, use -(N1+N2)
	innerDir := N1.Add(N2)
	if !isPositiveNormalSide {
		innerDir = innerDir.Mul(-1) // -N side inner → negate
	}

	innerDirLen := innerDir.Length()
	if innerDirLen < 1e-9 {
		return vec.Vec2{}, false
	}
	innerDir = innerDir.Mul(1 / innerDirLen)

	return P.Add(innerDir.Mul(d / halfAngle)), true
}

// addInnerIntersectionOrOffsets handles the inner side of a corner.
// If we can compute an intersection, adds just that point.
// Otherwise adds both offset points (fallback to current behavior).
// Returns true if intersection was used (next.A offset should be skipped).
func (r *Rasterizer) addInnerIntersectionOrOffsets(P, T1, T2, N1, N2 vec.Vec2, d float64, isPositiveNormalSide bool) bool {
	if innerPt, ok := computeInnerIntersection(P, T1, T2, d, isPositiveNormalSide); ok {
		r.stroke = append(r.stroke, innerPt)
		return true // skip next.A offset
	}
	// Fallback: add both offset points
	if isPositiveNormalSide {
		r.stroke = append(r.stroke, P.Add(N1.Mul(d)))
		r.stroke = append(r.stroke, P.Add(N2.Mul(d)))
	} else {
		r.stroke = append(r.stroke, P.Sub(N1.Mul(d)))
		r.stroke = append(r.stroke, P.Sub(N2.Mul(d)))
	}
	return false
}

// addJoin adds a line join at point P where tangent changes from T1 to T2.
// d is half the stroke width.
// isPositiveNormalSide indicates which side of the stroke we're building.
func (r *Rasterizer) addJoin(P, T1, T2 vec.Vec2, d float64, isPositiveNormalSide bool) {
	// Compute angle between tangents
	cosTheta := T1.Dot(T2)
	sinTheta := T1.X*T2.Y - T1.Y*T2.X // cross product Z component

	// Skip if nearly collinear
	if sinTheta > -collinearityThreshold && sinTheta < collinearityThreshold {
		return
	}

	// Check for cusp (path doubling back on itself)
	if cosTheta < cuspCosineThreshold {
		// Emit two caps instead of a join
		r.addCap(P, T1, d)
		r.addCap(P, T2.Mul(-1), d)
		return
	}

	// The join geometry extends in the direction of the current side we're building.
	// isPositiveNormalSide tells us which side: +N (true) or -N (false).

	switch r.Join {
	case graphics.LineJoinMiter:
		// Check miter limit: miterLength = 1 / sin(φ/2)
		// where φ is the visual angle at the corner (interior angle of the stroke).
		// If θ is the angle between tangents (cosTheta = T1·T2), then φ = 180° - θ.
		// sin(φ/2) = sin(90° − θ/2) = cos(θ/2) = sqrt((1 + cosθ) / 2)
		sinHalf := math.Sqrt((1 + cosTheta) / 2)
		// Use small tolerance for boundary cases (floating-point precision)
		const miterEpsilon = 1e-10
		if sinHalf > 0 && 1/sinHalf <= r.MiterLimit+miterEpsilon {
			// Miter join: compute miter point
			// The miter point is where the two offset lines intersect
			// Distance from P to miter point = d / sin(φ/2) = d / sinHalf
			N1 := vec.Vec2{X: -T1.Y, Y: T1.X}
			N2 := vec.Vec2{X: -T2.Y, Y: T2.X}

			// Bisector direction depends on which side we're building
			var bisector vec.Vec2
			if isPositiveNormalSide {
				bisector = N1.Add(N2) // +N side
			} else {
				bisector = N1.Add(N2).Mul(-1) // -N side
			}
			bisectorLen := bisector.Length()
			if bisectorLen > zeroLengthThreshold {
				bisector = bisector.Mul(1 / bisectorLen)
				// Distance to miter point = d / sinHalf
				miterDist := d / sinHalf
				miterPt := P.Add(bisector.Mul(miterDist))
				r.stroke = append(r.stroke, miterPt)
			}
			return
		}
		// Fall through to bevel if miter limit exceeded
		fallthrough

	case graphics.LineJoinBevel:
		// Bevel join: just let the two offset lines meet (no additional points)
		// The caller already adds the necessary points
		return

	case graphics.LineJoinRound:
		// Round join: arc curving outward on the current side
		// includeStart=false because join's start point is already in the polygon
		angle := math.Acos(max(-1, min(1, cosTheta)))
		if isPositiveNormalSide {
			// Forward pass: arc from +N of T1 to +N of T2
			N1 := vec.Vec2{X: -T1.Y, Y: T1.X} // +N direction of T1
			// For +N side: right turn needs CCW arc, left turn needs CW arc
			if sinTheta > 0 {
				r.addArc(P, d, N1, angle, false)
			} else {
				r.addArc(P, d, N1, -angle, false)
			}
		} else {
			// Backward pass: we just added offset using T2's normal, so arc must
			// start from -N of T2 and go to -N of T1 (reversed direction)
			N2 := vec.Vec2{X: T2.Y, Y: -T2.X} // -N direction of T2
			// Sweep direction is reversed from forward pass
			if sinTheta > 0 {
				r.addArc(P, d, N2, -angle, false)
			} else {
				r.addArc(P, d, N2, angle, false)
			}
		}
	}
}

// addArc adds arc vertices to the stroke outline.
// center is the arc center, radius is the arc radius.
// startDir is the unit vector from center to arc start.
// sweep is the sweep angle in radians (positive = CCW).
// includeStart indicates whether to include the start point (false if caller already added it).
func (r *Rasterizer) addArc(center vec.Vec2, radius float64, startDir vec.Vec2, sweep float64, includeStart bool) {
	// Compute number of segments based on flatness tolerance
	// Using device-space radius for segment count
	devRadius := r.transformLinear(vec.Vec2{X: radius, Y: 0}).Length()
	devRadius2 := r.transformLinear(vec.Vec2{X: 0, Y: radius}).Length()
	devRadius = max(devRadius, devRadius2)

	if devRadius < r.Flatness {
		// Arc too small to matter - just add end point (and start if needed)
		if includeStart {
			r.stroke = append(r.stroke, center.Add(startDir.Mul(radius)))
		}
		cos, sin := math.Cos(sweep), math.Sin(sweep)
		endDir := vec.Vec2{
			X: startDir.X*cos - startDir.Y*sin,
			Y: startDir.X*sin + startDir.Y*cos,
		}
		r.stroke = append(r.stroke, center.Add(endDir.Mul(radius)))
		return
	}

	// For a chord subtending angle θ on a circle of radius r, the maximum
	// deviation (sagitta) is r*(1 - cos(θ/2)). For this to equal tolerance ε:
	//   θ = 2*acos(1 - ε/r)
	// So for a sweep of S radians: n = ceil(S / θ) = ceil(S / (2*acos(1 - ε/r)))
	absSweep := math.Abs(sweep)

	angleStep := 2 * math.Acos(1-r.Flatness/devRadius)
	if angleStep <= 0 || math.IsNaN(angleStep) {
		angleStep = math.Pi / 4 // fallback
	}
	// cap before the int conversion so an extreme CTM cannot overflow it
	n := int(min(math.Ceil(absSweep/angleStep), maxFlattenSegments))
	n = max(n, 1)

	// Generate arc points
	dt := sweep / float64(n)
	startI := 0
	if !includeStart {
		startI = 1 // skip start point if caller already added it
	}
	for i := startI; i <= n; i++ {
		angle := float64(i) * dt
		// Rotate startDir by angle
		cos, sin := math.Cos(angle), math.Sin(angle)
		dir := vec.Vec2{
			X: startDir.X*cos - startDir.Y*sin,
			Y: startDir.X*sin + startDir.Y*cos,
		}
		pt := center.Add(dir.Mul(radius))
		r.stroke = append(r.stroke, pt)
	}
}

// addSquare adds a filled square to the stroke outline for a zero-length
// dash segment with square caps. The square is centered at the point with
// side length = 2*d (i.e., the line width), oriented by the tangent T.
func (r *Rasterizer) addSquare(center vec.Vec2, T vec.Vec2, d float64) {
	N := vec.Vec2{X: -T.Y, Y: T.X} // normal (90° CCW from T)
	// Four corners of the square
	r.stroke = append(r.stroke,
		center.Add(T.Mul(d)).Add(N.Mul(d)),
		center.Add(T.Mul(d)).Sub(N.Mul(d)),
		center.Sub(T.Mul(d)).Sub(N.Mul(d)),
		center.Sub(T.Mul(d)).Add(N.Mul(d)),
	)
}

// applyDashPattern applies the dash pattern to flattened subpaths.
// Results are stored in r.dashedSegs and r.dashedSegsOffsets.
func (r *Rasterizer) applyDashPattern() {
	// Clear output buffers (preserving capacity)
	r.dashedSegs = r.dashedSegs[:0]
	r.dashedSegsOffsets = r.dashedSegsOffsets[:0]

	dash := r.Dash
	dashLen := len(dash)

	// Compute total pattern length (doubled for odd-length patterns)
	rawSum := 0.0
	for _, d := range dash {
		rawSum += d
	}
	patternLen := rawSum
	if dashLen%2 == 1 {
		patternLen *= 2
	}
	if patternLen <= 0 {
		return // no dashing
	}

	// Normalize phase to [0, patternLen)
	phase := r.DashPhase
	phase = math.Mod(phase, patternLen)
	if phase < 0 {
		phase += patternLen
	}

	numSubpaths := len(r.segsOffsets)
	for spIdx := range numSubpaths {
		segments := r.getSubpathSegments(spIdx)
		closed := r.subpathClosed[spIdx]
		if len(segments) == 0 {
			continue
		}

		// guard against a pathologically fine dash pattern: splitting this
		// subpath into more than maxDashSegments pieces would be sub-pixel on
		// any reasonable canvas, so stroke it solid instead of hanging. The dash
		// loop consumes one element per piece, so the piece count is
		// subpathLen/patternLen periods times the elements per period; dividing
		// by the un-doubled sum folds in the odd-length doubling of patternLen.
		// A closed subpath taken down this path gets caps rather than a closing
		// join, but only in this invisible regime.
		subpathLen := 0.0
		for _, seg := range segments {
			subpathLen += seg.B.Sub(seg.A).Length()
		}
		if subpathLen*float64(dashLen)/rawSum > float64(maxDashSegments) {
			dashStart := len(r.dashedSegs)
			r.dashedSegs = append(r.dashedSegs, segments...)
			r.dashedSegsOffsets = append(r.dashedSegsOffsets, dashStart)
			continue
		}

		// starting dash index and remaining distance in that dash,
		// advancing past zero-length dash elements while consuming phase
		dashIdx := 0
		dist := phase
		for dist > 0 && dist >= dash[dashIdx%dashLen] {
			dist -= dash[dashIdx%dashLen]
			dashIdx++
		}
		remaining := dash[dashIdx%dashLen] - dist
		isOn := dashIdx%2 == 0 // even indices are "on"

		// Handle zero-length dash at the very start of the path.
		// This emits a point that will become a dot with round/square caps.
		if isOn && remaining == 0 && len(segments) > 0 {
			seg := segments[0]
			r.dashedSegsOffsets = append(r.dashedSegsOffsets, len(r.dashedSegs))
			r.dashedSegs = append(r.dashedSegs, strokeSegment{A: seg.A, B: seg.A, T: seg.T, N: seg.N})
			// Advance to next dash element
			dashIdx++
			remaining = dash[dashIdx%dashLen]
			isOn = dashIdx%2 == 0
		}

		// Track if we started with "on" for closed path joining
		startedOn := isOn
		firstDashStart := -1 // index into dashedSegs where first dash starts
		firstDashEnd := -1   // index into dashedSegs where first dash ends

		// Walk segments and split at dash boundaries
		dashStartIdx := len(r.dashedSegs) // start of current dash in dashedSegs
		segIdx := 0
		segDist := 0.0 // distance along current segment

		for segIdx < len(segments) {
			seg := segments[segIdx]
			segLen := seg.B.Sub(seg.A).Length()
			segRemaining := segLen - segDist

			if remaining >= segRemaining {
				// Dash continues past this segment
				if isOn {
					// Add portion of segment from segDist to end
					if segDist > 0 {
						t := segDist / segLen
						startPt := seg.A.Add(seg.B.Sub(seg.A).Mul(t))
						r.dashedSegs = append(r.dashedSegs, strokeSegment{
							A: startPt, B: seg.B,
							T: seg.T, N: seg.N,
						})
					} else {
						r.dashedSegs = append(r.dashedSegs, seg)
					}
				}
				remaining -= segRemaining
				segIdx++
				segDist = 0
			} else {
				// Dash ends within this segment
				endDist := segDist + remaining
				t := endDist / segLen
				splitPt := seg.A.Add(seg.B.Sub(seg.A).Mul(t))

				if isOn {
					// Add portion from segDist to splitPt
					startT := segDist / segLen
					startPt := seg.A.Add(seg.B.Sub(seg.A).Mul(startT))
					d := splitPt.Sub(startPt)
					dLen := d.Length()
					if dLen > zeroLengthThreshold {
						tVec := d.Mul(1 / dLen)
						nVec := vec.Vec2{X: -tVec.Y, Y: tVec.X}
						r.dashedSegs = append(r.dashedSegs, strokeSegment{
							A: startPt, B: splitPt,
							T: tVec, N: nVec,
						})
					} else if len(r.dashedSegs) == dashStartIdx {
						// Zero-length dash: emit point with tangent from underlying segment
						// This allows square/round caps to be drawn at this point
						r.dashedSegs = append(r.dashedSegs, strokeSegment{
							A: startPt, B: startPt,
							T: seg.T, N: seg.N,
						})
					}

					// Save first dash indices for closed path joining
					if firstDashStart < 0 && len(r.dashedSegs) > dashStartIdx {
						firstDashStart = dashStartIdx
						firstDashEnd = len(r.dashedSegs)
					}

					// Emit current dash if non-empty
					if len(r.dashedSegs) > dashStartIdx {
						r.dashedSegsOffsets = append(r.dashedSegsOffsets, dashStartIdx)
						dashStartIdx = len(r.dashedSegs)
					}
				}

				// Move to next dash
				segDist = endDist
				dashIdx++
				remaining = dash[dashIdx%dashLen]
				isOn = dashIdx%2 == 0
			}
		}

		// Emit final dash if any
		if len(r.dashedSegs) > dashStartIdx {
			// For closed paths, check if we should join first and last dash
			if closed && startedOn && isOn && firstDashStart >= 0 {
				// Merge: append first dash segments to current dash
				for i := firstDashStart; i < firstDashEnd; i++ {
					r.dashedSegs = append(r.dashedSegs, r.dashedSegs[i])
				}
				// Remove the first dash from offsets if we added it
				if len(r.dashedSegsOffsets) > 0 && r.dashedSegsOffsets[0] == firstDashStart {
					r.dashedSegsOffsets = r.dashedSegsOffsets[1:]
				}
			}
			r.dashedSegsOffsets = append(r.dashedSegsOffsets, dashStartIdx)
		}
	}
}

// fillStrokeOutlines fills all collected stroke polygons as a compound path.
// Using nonzero winding rule ensures overlapping regions are painted once.
func (r *Rasterizer) fillStrokeOutlines(emit func(y, xMin int, coverage []float32)) {
	if len(r.strokeOffsets) == 0 {
		return
	}

	// Collect edges directly from stroke polygons (no intermediate path allocation)
	xMin, xMax, yMin, yMax, ok := r.collectStrokeEdges()
	if !ok {
		return
	}

	// Choose approach based on bounding box size
	width := xMax - xMin
	height := yMax - yMin

	if width*height < r.smallPathThreshold {
		r.fillSmallPath(xMin, xMax, yMin, yMax, fillNonZero, emit)
	} else {
		r.fillLargePath(xMin, xMax, yMin, yMax, fillNonZero, emit)
	}
}

// collectStrokeEdges builds the edge list directly from stroke polygons.
// This avoids creating an intermediate path representation.
func (r *Rasterizer) collectStrokeEdges() (xMin, xMax, yMin, yMax int, ok bool) {
	r.edges = r.edges[:0]
	r.edgeBBoxFirst = true

	for i, start := range r.strokeOffsets {
		// Determine end of this polygon
		var end int
		if i+1 < len(r.strokeOffsets) {
			end = r.strokeOffsets[i+1]
		} else {
			end = len(r.stroke)
		}
		poly := r.stroke[start:end]
		if len(poly) < 2 {
			continue
		}

		// Add edges for each segment
		for j := 1; j < len(poly); j++ {
			r.addEdge(poly[j-1], poly[j])
		}
		// Close the polygon
		r.addEdge(poly[len(poly)-1], poly[0])
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
