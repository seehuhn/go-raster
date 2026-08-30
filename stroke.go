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
	"slices"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

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
	if r.Clip.Empty() {
		return // skip flattening and edge collection when nothing can be emitted
	}

	// bound on the CTM's device-space scaling, used for join folding and arc
	// segment counts
	_, r.ctmSigmaMax = r.CTM.SingularValues()

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
			start := r.beginPiece()
			// negative sweep, so the circle winds clockwise like every other piece
			r.addArc(pt, r.Width/2, vec.Vec2{X: 1, Y: 0}, -2*math.Pi)
			r.endPiece(start)
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
		r.strokeSubpath(r.getSubpathSegments(i), r.subpathClosed[i])
	}
}

// strokeSegment represents a line segment in user coordinates
type strokeSegment struct {
	A, B vec.Vec2 // endpoints in user space
	T    vec.Vec2 // unit tangent (A→B direction)
	N    vec.Vec2 // unit normal (90° CCW from T)
	Len  float64  // distance from A to B
}

// getSubpathSegments returns the segments for subpath i as a slice into segs.
func (r *Rasterizer) getSubpathSegments(i int) []strokeSegment {
	return sliceAt(r.segs, r.segsOffsets, i)
}

// sliceAt returns slice i of data, where offsets holds the start index of
// each slice and the last slice runs to the end of data.
func sliceAt[T any](data []T, offsets []int, i int) []T {
	start := offsets[i]
	end := len(data)
	if i+1 < len(offsets) {
		end = offsets[i+1]
	}
	return data[start:end]
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
			start := r.beginPiece()
			switch r.Cap {
			case graphics.LineCapRound:
				// negative sweep, so the circle winds clockwise like every other piece
				r.addArc(seg.A, r.Width/2, vec.Vec2{X: 1, Y: 0}, -2*math.Pi)
			case graphics.LineCapSquare:
				r.addSquare(seg.A, seg.T, r.Width/2)
			}
			// Butt cap: no output
			r.endPiece(start)
			continue
		}

		r.strokeSubpath(segs, r.dashedClosed[i])
	}
}

// getDashedSegments returns the segments for dashed subpath i as a slice into dashedSegs.
func (r *Rasterizer) getDashedSegments(i int) []strokeSegment {
	return sliceAt(r.dashedSegs, r.dashedSegsOffsets, i)
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
				r.finishSubpath(subpathStartIdx, subpathStartPt, false)
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
				r.finishSubpath(subpathStartIdx, subpathStartPt, true)
				currentPt = subpathStartPt
				subpathStartIdx = len(r.segs)
				inSubpath = false
				sawDrawingCmd = false
			}
		}
	}

	// handle unclosed subpath at end
	if inSubpath && (len(r.segs) > subpathStartIdx || sawDrawingCmd) {
		r.finishSubpath(subpathStartIdx, subpathStartPt, false)
	}
}

// finishSubpath records the subpath starting at startIdx in r.segs: as a
// degenerate point (no orientation) if it has no segments, otherwise as a
// regular subpath with the given closedness.
func (r *Rasterizer) finishSubpath(startIdx int, startPt vec.Vec2, closed bool) {
	if len(r.segs) == startIdx {
		r.degeneratePoints = append(r.degeneratePoints, startPt)
	} else {
		r.segsOffsets = append(r.segsOffsets, startIdx)
		r.subpathClosed = append(r.subpathClosed, closed)
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
	r.segs = append(r.segs, strokeSegment{A: a, B: b, T: t, N: n, Len: length})
}

// beginPiece returns the index at which the next piece of the stroke outline
// starts.
func (r *Rasterizer) beginPiece() int {
	return len(r.stroke)
}

// endPiece records the points added since start as one piece of the stroke
// outline, dropping pieces with too few points to cover any area.
//
// Every piece must wind clockwise.  Pieces of the same stroke overlap freely,
// and only pieces which agree on their winding direction reinforce each other
// under the nonzero winding rule; a piece wound the other way would cancel the
// pieces it overlaps and leave a hole.
func (r *Rasterizer) endPiece(start int) {
	if len(r.stroke)-start >= 3 {
		r.strokeOffsets = append(r.strokeOffsets, start)
	} else {
		r.stroke = r.stroke[:start]
	}
}

// strokeSubpath builds the stroke outline for a single subpath into r.stroke.
//
// The outline is emitted as a union of simple clockwise pieces.  Runs of
// segments whose joints turn gently merge into strips: one polygon walking
// forward along the left offsets and back along the right offsets, with a
// short chord standing in for each interior join.  A joint that turns too
// sharply to fold breaks the strip and receives a separate join piece, so a
// strip degenerates to one quadrilateral per segment where the path has real
// corners.  Every piece winds the same way, so the nonzero winding rule fills
// exactly their union, however much the stroke overlaps itself.
//
// Zero-length subpaths are handled by the caller before invoking this method.
func (r *Rasterizer) strokeSubpath(segs []strokeSegment, closed bool) {
	n := len(segs)
	if n == 0 {
		return
	}

	d := r.Width / 2

	// device-space bound on the half width, used in the folding tolerance
	dDev := d * r.ctmSigmaMax

	// foldable reports whether the joint between prev and cur can be folded
	// into a strip, and returns an upper bound on the turn angle there.
	//
	// Folding replaces the join by the chord between the two offset points.
	// The first test bounds the substitution error, which is below d*sin²θ
	// for every join style, by the flatness tolerance.  The second test is
	// the winding-safety condition: the offset distance must stay below the
	// local radius of curvature, with the margin split between the two ends
	// of each segment, so the inner boundary keeps advancing and the strip
	// stays simple.
	foldable := func(prev, cur *strokeSegment) (bool, float64) {
		cosT := prev.T.Dot(cur.T)
		if cosT <= 0 {
			return false, 0
		}
		sinT := prev.T.X*cur.T.Y - prev.T.Y*cur.T.X
		if dDev*sinT*sinT > r.Flatness {
			return false, 0
		}
		tanHalf := math.Abs(sinT) / (1 + cosT)
		if d*tanHalf > 0.49*min(prev.Len, cur.Len) {
			return false, 0
		}
		return true, 2 * tanHalf // 2*tan(θ/2) ≥ θ
	}

	// emitStrip emits count segments starting at index first, wrapping around
	// for closed subpaths, as one clockwise piece.
	emitStrip := func(first, count int) {
		start := r.beginPiece()
		seg := &segs[first]
		r.stroke = append(r.stroke, seg.A.Add(seg.N.Mul(d)))
		for k := range count {
			seg = &segs[(first+k)%n]
			r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
			if k+1 < count {
				next := &segs[(first+k+1)%n]
				r.stroke = append(r.stroke, next.A.Add(next.N.Mul(d)))
			}
		}
		for k := count - 1; k >= 0; k-- {
			seg = &segs[(first+k)%n]
			r.stroke = append(r.stroke, seg.B.Sub(seg.N.Mul(d)), seg.A.Sub(seg.N.Mul(d)))
		}
		r.endPiece(start)
	}

	if !closed {
		stripStart := 0
		turn := 0.0
		for i := 1; i < n; i++ {
			prev, cur := &segs[i-1], &segs[i]
			ok, dTurn := foldable(prev, cur)
			if ok && turn+dTurn <= maxStripTurn {
				turn += dTurn
				continue
			}
			emitStrip(stripStart, i-stripStart)
			r.addJoinPiece(cur.A, prev.T, cur.T, d)
			stripStart = i
			turn = 0
		}
		emitStrip(stripStart, n-stripStart)

		r.addCapPiece(segs[0].A, segs[0].T.Mul(-1), d)
		r.addCapPiece(segs[n-1].B, segs[n-1].T, d)
		return
	}

	// A closed subpath has a joint before every segment, including the seam
	// between the last segment and the first.  Strips may cross the seam, so
	// start at a joint that must break anyway: the first unfoldable joint, or
	// joint 0 if the whole loop is gentle (the turn cap, which any closed
	// loop exceeds, then forces breaks elsewhere).
	breakAt := 0
	for j := range n {
		prev := &segs[(j+n-1)%n]
		if ok, _ := foldable(prev, &segs[j]); !ok {
			breakAt = j
			break
		}
	}

	stripStart := breakAt
	turn := 0.0
	for k := 1; k < n; k++ {
		i := (breakAt + k) % n
		prev, cur := &segs[(i+n-1)%n], &segs[i]
		ok, dTurn := foldable(prev, cur)
		if ok && turn+dTurn <= maxStripTurn {
			turn += dTurn
			continue
		}
		emitStrip(stripStart, (i-stripStart+n)%n)
		r.addJoinPiece(cur.A, prev.T, cur.T, d)
		stripStart = i
		turn = 0
	}
	emitStrip(stripStart, (breakAt-stripStart-1+n)%n+1)
	prev := &segs[(breakAt+n-1)%n]
	r.addJoinPiece(segs[breakAt].A, prev.T, segs[breakAt].T, d)
}

// addJoinPiece adds the join at P as a separate piece of the stroke outline.
// T1 and T2 are the tangents of the incoming and outgoing segment, d is half
// the stroke width.
func (r *Rasterizer) addJoinPiece(P, T1, T2 vec.Vec2, d float64) {
	cosTheta := max(-1, min(1, T1.Dot(T2)))
	sinTheta := T1.X*T2.Y - T1.Y*T2.X

	// The signed turn angle, in (-pi, pi].  Taking it from Atan2 rather than
	// from the sign of sinTheta alone keeps a cusp well defined: there sinTheta
	// is zero, but Atan2 still yields +/-pi and picks out the side the path
	// doubles back towards.
	turn := math.Atan2(sinTheta, cosTheta)

	// Where the path continues straight the two quadrilaterals meet flush and
	// there is no wedge to fill.  The bound is numerical, not a quality
	// tolerance: no point of the omitted sector lies further than d*sin(turn/2)
	// from one of the two quadrilaterals, and at this angle that is under 1e-6
	// times the line width, whatever the width and the CTM.
	if math.Abs(turn) < collinearityThreshold {
		return
	}

	// The join sits on the outer side of the corner, away from the turn, and
	// the outer normals span exactly the turn angle.
	outer := -math.Copysign(1, turn)
	n1 := vec.Vec2{X: -T1.Y, Y: T1.X}.Mul(outer)
	n2 := vec.Vec2{X: -T2.Y, Y: T2.X}.Mul(outer)

	// Every piece must wind the same way as the segment quadrilaterals, which
	// run clockwise, so sweep from whichever normal leaves a clockwise arc.
	from, to, sweep := n1, n2, turn
	if sweep > 0 {
		from, to, sweep = n2, n1, -turn
	}

	switch r.Join {
	case graphics.LineJoinMiter:
		// miterLength = 1/sin(φ/2), where φ = 180° - θ is the interior angle at
		// the corner, so sin(φ/2) = cos(θ/2) = sqrt((1 + cosθ)/2).
		sinHalf := math.Sqrt((1 + cosTheta) / 2)
		const miterEpsilon = 1e-10
		if sinHalf > 0 && 1/sinHalf <= r.MiterLimit+miterEpsilon {
			bisector := n1.Add(n2)
			if bisectorLen := bisector.Length(); bisectorLen > zeroLengthThreshold {
				tip := P.Add(bisector.Mul(d / (sinHalf * bisectorLen)))
				start := r.beginPiece()
				r.stroke = append(r.stroke, P, P.Add(from.Mul(d)), tip, P.Add(to.Mul(d)))
				r.endPiece(start)
				return
			}
		}
		// miter limit exceeded, fall back to a bevel
		fallthrough

	case graphics.LineJoinBevel:
		start := r.beginPiece()
		r.stroke = append(r.stroke, P, P.Add(from.Mul(d)), P.Add(to.Mul(d)))
		r.endPiece(start)

	case graphics.LineJoinRound:
		start := r.beginPiece()
		r.stroke = append(r.stroke, P)
		r.addArc(P, d, from, sweep)
		r.endPiece(start)
	}
}

// addCapPiece adds the cap at P as a separate piece of the stroke outline.
// T is the outward tangent, pointing away from the line, and d is half the
// stroke width.
func (r *Rasterizer) addCapPiece(P, T vec.Vec2, d float64) {
	N := vec.Vec2{X: -T.Y, Y: T.X} // normal (90° CCW from T)

	switch r.Cap {
	case graphics.LineCapButt:
		// nothing to add: the segment quadrilateral already ends flush

	case graphics.LineCapSquare:
		ext := T.Mul(d)
		start := r.beginPiece()
		r.stroke = append(r.stroke,
			P.Add(N.Mul(d)), P.Add(N.Mul(d)).Add(ext),
			P.Sub(N.Mul(d)).Add(ext), P.Sub(N.Mul(d)))
		r.endPiece(start)

	case graphics.LineCapRound:
		// semicircle from +N clockwise through T to -N
		start := r.beginPiece()
		r.addArc(P, d, N, -math.Pi)
		r.endPiece(start)
	}
}

// addArc adds arc vertices to the stroke outline.
// center is the arc center, radius is the arc radius.
// startDir is the unit vector from center to arc start.
// sweep is the sweep angle in radians (positive = CCW).
func (r *Rasterizer) addArc(center vec.Vec2, radius float64, startDir vec.Vec2, sweep float64) {
	// device-space radius bound, used for segment count
	devRadius := radius * r.ctmSigmaMax

	if devRadius < r.Flatness {
		// arc too small to matter, just add its endpoints
		r.stroke = append(r.stroke, center.Add(startDir.Mul(radius)))
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
	for i := 0; i <= n; i++ {
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
// Results are stored in r.dashedSegs, r.dashedSegsOffsets and
// r.dashedClosed.  A dash is closed only if it covers a whole closed
// subpath; a dash which wraps around the start point of a closed subpath
// is merged with the first dash into one open dash.
func (r *Rasterizer) applyDashPattern() {
	// Clear output buffers (preserving capacity)
	r.dashedSegs = r.dashedSegs[:0]
	r.dashedSegsOffsets = r.dashedSegsOffsets[:0]
	r.dashedClosed = r.dashedClosed[:0]

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
			subpathLen += seg.Len
		}
		if subpathLen*float64(dashLen)/rawSum > float64(maxDashSegments) {
			dashStart := len(r.dashedSegs)
			r.dashedSegs = append(r.dashedSegs, segments...)
			r.dashedSegsOffsets = append(r.dashedSegsOffsets, dashStart)
			r.dashedClosed = append(r.dashedClosed, closed)
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
			r.dashedClosed = append(r.dashedClosed, false)
			r.dashedSegs = append(r.dashedSegs, strokeSegment{A: seg.A, B: seg.A, T: seg.T, N: seg.N})
			// Advance to next dash element
			dashIdx++
			remaining = dash[dashIdx%dashLen]
			isOn = dashIdx%2 == 0
		}

		// Track if we started with "on" for closed path joining
		startedOn := isOn
		firstDashStart := -1  // index into dashedSegs where first dash starts
		firstDashOffset := -1 // index into dashedSegsOffsets of the first dash

		// Walk segments and split at dash boundaries
		dashStartIdx := len(r.dashedSegs) // start of current dash in dashedSegs
		segIdx := 0
		segDist := 0.0 // distance along current segment

		for segIdx < len(segments) {
			seg := segments[segIdx]
			segLen := seg.Len
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
							Len: segRemaining,
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
							Len: dLen,
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
						firstDashOffset = len(r.dashedSegsOffsets)
					}

					// Emit current dash if non-empty
					if len(r.dashedSegs) > dashStartIdx {
						r.dashedSegsOffsets = append(r.dashedSegsOffsets, dashStartIdx)
						r.dashedClosed = append(r.dashedClosed, false)
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
			if closed && startedOn && isOn && firstDashStart >= 0 {
				// The pattern wraps around the start point, so the final
				// dash continues into the first one.  Move the final dash's
				// segments in front of the first dash, making the merged
				// dash contiguous, and shift the offsets in between.
				segs := r.dashedSegs[firstDashStart:]
				lastLen := len(r.dashedSegs) - dashStartIdx
				slices.Reverse(segs)
				slices.Reverse(segs[:lastLen])
				slices.Reverse(segs[lastLen:])
				for j := firstDashOffset + 1; j < len(r.dashedSegsOffsets); j++ {
					r.dashedSegsOffsets[j] += lastLen
				}
				continue
			}
			r.dashedSegsOffsets = append(r.dashedSegsOffsets, dashStartIdx)
			// the dash is closed only if it covers the whole closed subpath
			r.dashedClosed = append(r.dashedClosed, closed && startedOn && isOn)
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

	r.rasterizeEdges(xMin, xMax, yMin, yMax, fillNonZero, emit)
}

// collectStrokeEdges builds the edge list directly from stroke polygons.
// This avoids creating an intermediate path representation.
func (r *Rasterizer) collectStrokeEdges() (xMin, xMax, yMin, yMax int, ok bool) {
	r.edges = r.edges[:0]
	r.edgeBBoxFirst = true

	for i := range r.strokeOffsets {
		poly := sliceAt(r.stroke, r.strokeOffsets, i)
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

	return r.clippedEdgeBounds()
}
