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
	"testing"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
)

// An attacker-controlled CTM scale must not let a single curve flatten into an
// unbounded number of segments.

func TestFlattenCubicSegmentCap(t *testing.T) {
	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.CTM = matrix.Matrix{1e10, 0, 0, 1e10, 0, 0}

	count := 0
	r.flattenCubic(
		vec.Vec2{X: 0, Y: 0},
		vec.Vec2{X: 0, Y: 1},
		vec.Vec2{X: 1, Y: 0},
		vec.Vec2{X: 1, Y: 1},
		func(from, to vec.Vec2) { count++ },
	)
	if count > maxFlattenSegments {
		t.Errorf("cubic flattened into %d segments, want <= %d", count, maxFlattenSegments)
	}
}

func TestFlattenQuadraticSegmentCap(t *testing.T) {
	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.CTM = matrix.Matrix{1e10, 0, 0, 1e10, 0, 0}

	count := 0
	r.flattenQuadratic(
		vec.Vec2{X: 0, Y: 0},
		vec.Vec2{X: 0, Y: 1},
		vec.Vec2{X: 1, Y: 0},
		func(from, to vec.Vec2) { count++ },
	)
	if count > maxFlattenSegments {
		t.Errorf("quadratic flattened into %d segments, want <= %d", count, maxFlattenSegments)
	}
}

func TestArcSegmentCap(t *testing.T) {
	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.CTM = matrix.Matrix{1e10, 0, 0, 1e10, 0, 0}
	r.Width = 1

	r.stroke = r.stroke[:0]
	r.addArc(vec.Vec2{X: 0, Y: 0}, 1, vec.Vec2{X: 1, Y: 0}, 2*math.Pi, true)
	if len(r.stroke) > maxFlattenSegments+1 {
		t.Errorf("arc generated %d vertices, want <= %d", len(r.stroke), maxFlattenSegments+1)
	}
}

// The cap must not perturb flattening at ordinary scales.

func TestFlattenNormalScaleUnaffected(t *testing.T) {
	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.CTM = matrix.Identity

	count := 0
	r.flattenCubic(
		vec.Vec2{X: 0, Y: 0},
		vec.Vec2{X: 0, Y: 50},
		vec.Vec2{X: 50, Y: 0},
		vec.Vec2{X: 50, Y: 50},
		func(from, to vec.Vec2) { count++ },
	)
	if count == 0 || count > 1000 {
		t.Errorf("cubic at scale 1 flattened into %d segments, want a small positive number", count)
	}
}

// A non-finite or extreme CTM must not panic, hang, or produce out-of-range
// device coordinates; the result is a safely-clipped (empty) raster.

func TestFillExtremeCTM(t *testing.T) {
	p := path.Data{}
	p.MoveTo(vec.Vec2{X: 0, Y: 0})
	p.CubeTo(vec.Vec2{X: 0, Y: 1}, vec.Vec2{X: 1, Y: 0}, vec.Vec2{X: 1, Y: 1})
	p.Close()
	pth := p.Iter()

	scales := []float64{1e18, 1e150, 1e300, math.Inf(1), math.NaN()}
	for _, s := range scales {
		r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
		r.CTM = matrix.Matrix{s, 0, 0, s, 0, 0}
		// must complete without panic or hang
		r.FillNonZero(pth, func(y, xMin int, coverage []float32) {})
	}
}

// Stroking shares the extreme-CTM hazard: the stroke outline is built in user
// space and only transformed to device coordinates when its edges are
// collected, so the same non-finite/out-of-range bounds must be handled there.

func TestStrokeExtremeCTM(t *testing.T) {
	p := path.Data{}
	p.MoveTo(vec.Vec2{X: 0, Y: 0})
	p.CubeTo(vec.Vec2{X: 0, Y: 1}, vec.Vec2{X: 1, Y: 0}, vec.Vec2{X: 1, Y: 1})
	p.Close()
	pth := p.Iter()

	scales := []float64{1e18, 1e150, 1e300, math.Inf(1), math.NaN()}
	for _, s := range scales {
		r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
		r.CTM = matrix.Matrix{s, 0, 0, s, 0, 0}
		r.Width = 1
		// must complete without panic or hang
		r.Stroke(pth, func(y, xMin int, coverage []float32) {})
	}
}

// An attacker-controlled dash array must not split a path into an unbounded
// number of pieces; a pattern that fine is rendered solid instead.

func TestDashSegmentCap(t *testing.T) {
	p := path.Data{}
	p.MoveTo(vec.Vec2{X: 0, Y: 0})
	p.LineTo(vec.Vec2{X: 1e6, Y: 0})

	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.Width = 1
	r.Dash = []float64{0.001}
	// must complete without hanging
	r.Stroke(p.Iter(), func(y, xMin int, coverage []float32) {})

	if len(r.dashedSegs) > maxDashSegments {
		t.Errorf("dashing produced %d segments, want <= %d", len(r.dashedSegs), maxDashSegments)
	}
}

// A pattern dominated by zero-length elements has a tiny total length but many
// elements per period; the piece count must still be bounded by the elements
// consumed, not by the smallest positive element alone.

func TestDashZeroPaddedCap(t *testing.T) {
	dash := make([]float64, 1000)
	dash[0] = 5
	// 999 trailing zeros: minDash would be 5, but each period emits ~500 dots

	p := path.Data{}
	p.MoveTo(vec.Vec2{X: 0, Y: 0})
	p.LineTo(vec.Vec2{X: 327000, Y: 0})

	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.Width = 1
	r.Dash = dash
	// must complete without hanging
	r.Stroke(p.Iter(), func(y, xMin int, coverage []float32) {})

	if len(r.dashedSegs) > maxDashSegments {
		t.Errorf("dashing produced %d segments, want <= %d", len(r.dashedSegs), maxDashSegments)
	}
}

// The cap must not collapse an ordinary dash pattern into a solid stroke.

func TestDashNormalUnaffected(t *testing.T) {
	p := path.Data{}
	p.MoveTo(vec.Vec2{X: 0, Y: 0})
	p.LineTo(vec.Vec2{X: 50, Y: 0})

	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.Width = 1
	r.Dash = []float64{5, 3}
	r.Stroke(p.Iter(), func(y, xMin int, coverage []float32) {})

	if len(r.dashedSegsOffsets) <= 1 {
		t.Errorf("normal dash pattern produced %d dashes, want > 1", len(r.dashedSegsOffsets))
	}
}

// A long path with a sparse pattern produces many dashes but few pieces; the
// cap must not mistake its length for fineness and collapse it to solid.

func TestDashLongSparseUnaffected(t *testing.T) {
	p := path.Data{}
	p.MoveTo(vec.Vec2{X: 0, Y: 0})
	p.LineTo(vec.Vec2{X: 70000, Y: 0})

	r := NewRasterizer(rect.Rect{LLx: 0, LLy: 0, URx: 100, URy: 100})
	r.Width = 1
	r.Dash = []float64{1, 99} // ~700 dashes, far below the cap
	r.Stroke(p.Iter(), func(y, xMin int, coverage []float32) {})

	if len(r.dashedSegsOffsets) <= 1 {
		t.Errorf("long sparse dash pattern collapsed to %d dashes, want many", len(r.dashedSegsOffsets))
	}
	if len(r.dashedSegs) > maxDashSegments {
		t.Errorf("dashing produced %d segments, want <= %d", len(r.dashedSegs), maxDashSegments)
	}
}
