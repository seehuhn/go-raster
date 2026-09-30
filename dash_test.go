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
	"fmt"
	"image"
	"math"
	"testing"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// appendSquare adds a closed square subpath with corner (x, y) and side s.
func appendSquare(p *path.Data, x, y, s float64) {
	p.MoveTo(vec.Vec2{X: x, Y: y})
	p.LineTo(vec.Vec2{X: x + s, Y: y})
	p.LineTo(vec.Vec2{X: x + s, Y: y + s})
	p.LineTo(vec.Vec2{X: x, Y: y + s})
	p.Close()
}

// compareGrids fails the test where two coverage grids differ.
func compareGrids(t *testing.T, want, got [][]float32) {
	t.Helper()
	for y := range want {
		for x := range want[y] {
			if math.Abs(float64(want[y][x]-got[y][x])) > 1e-5 {
				t.Fatalf("pixel (%d,%d): coverage %g, want %g", x, y, got[y][x], want[y][x])
			}
		}
	}
}

// TestDashClosedSingleDash checks that a closed subpath which lies entirely
// within one dash is stroked exactly like the undashed subpath, with a join
// rather than caps at its start point.
func TestDashClosedSingleDash(t *testing.T) {
	p := &path.Data{}
	appendSquare(p, 20, 20, 50)

	solid := NewRasterizer(image.Rect(0, 0, 100, 100))
	solid.Width = 10
	want := renderGrid(t, solid, func(emit func(int, int, []float32)) { solid.Stroke(p.Iter(), emit) })

	dashed := NewRasterizer(image.Rect(0, 0, 100, 100))
	dashed.Width = 10
	dashed.Dash = []float64{1000, 10}
	got := renderGrid(t, dashed, func(emit func(int, int, []float32)) { dashed.Stroke(p.Iter(), emit) })

	compareGrids(t, want, got)
}

// TestDashClosedWrapSubpaths checks that when the dash pattern wraps around
// the start point of a closed subpath, the first and last dash merge into
// one for every subpath, not only the first: with square caps and round
// joins a leftover cap at the seam would protrude beyond the join.
func TestDashClosedWrapSubpaths(t *testing.T) {
	setup := func() *Rasterizer {
		r := NewRasterizer(image.Rect(0, 0, 100, 100))
		r.Width = 4
		r.Cap = path.CapSquare
		r.Join = path.JoinRound
		r.Dash = []float64{25, 5}
		r.DashPhase = 10 // the last dash runs on through the start point
		return r
	}
	a := &path.Data{}
	appendSquare(a, 10, 10, 30)
	b := &path.Data{}
	appendSquare(b, 60, 60, 30)
	ab := &path.Data{}
	appendSquare(ab, 10, 10, 30)
	appendSquare(ab, 60, 60, 30)

	r := setup()
	gridA := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(a.Iter(), emit) })
	r = setup()
	gridB := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(b.Iter(), emit) })
	r = setup()
	gridAB := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(ab.Iter(), emit) })

	// the squares do not overlap, so the union is the sum
	for y := range gridA {
		for x := range gridA[y] {
			gridA[y][x] += gridB[y][x]
		}
	}
	compareGrids(t, gridA, gridAB)
}

// TestDashBoundaryOnVertex checks dash boundaries which fall on a corner of
// the path.  A dash ending there takes its cap from the segment before the
// corner, a dash starting there from the segment after it, and a
// zero-length dash there from the segment before it.  The path is also
// placed far from the origin, where the arc lengths computed for the
// boundaries and the vertex differ by rounding, on either side of the
// corner; the result must not depend on it.
func TestDashBoundaryOnVertex(t *testing.T) {
	// a polyline turning at arc length 0.9, into a 3-4-5 diagonal
	const turnAt = 0.9
	dir2 := vec.Vec2{X: 0.6, Y: 0.8}
	at := func(o vec.Vec2, s float64) vec.Vec2 {
		if s <= turnAt {
			return o.Add(vec.Vec2{X: s})
		}
		return o.Add(vec.Vec2{X: turnAt}).Add(dir2.Mul(s - turnAt))
	}

	type dashCase struct {
		name  string
		cap   path.CapStyle
		dash  []float64
		phase float64
		dots  bool // the dashes have zero length
	}
	cases := []dashCase{
		{name: "dash ends on corner", cap: path.CapRound, dash: []float64{0.1, 0.1}},
		{name: "dash starts on corner", cap: path.CapRound, dash: []float64{0.1, 0.1}, phase: 0.1},
		{name: "zero-length dash on corner", cap: path.CapSquare, dash: []float64{0, 0.3}, dots: true},
		// twice 0.45 is exactly 0.9, so that the dash lands on the corner
		// without rounding
		{name: "zero-length dash exactly on corner", cap: path.CapSquare, dash: []float64{0, 0.45}, dots: true},
	}

	const width = 0.05
	for _, tc := range cases {
		for _, offset := range []float64{0, 1e7, 2e7} {
			t.Run(fmt.Sprintf("%s/offset=%g", tc.name, offset), func(t *testing.T) {
				o := vec.Vec2{X: offset, Y: offset}
				setup := func() *Rasterizer {
					r := NewRasterizer(image.Rect(0, 0, 170, 170))
					r.CTM = matrix.Matrix{100, 0, 0, 100, 10 - 100*offset, 10 - 100*offset}
					r.Width = width
					r.Cap = tc.cap
					r.Join = path.JoinMiter
					r.MiterLimit = 10
					return r
				}

				p := (&path.Data{}).MoveTo(at(o, 0))
				p.LineTo(at(o, turnAt))
				p.LineTo(at(o, 2*turnAt))
				r := setup()
				r.Dash = tc.dash
				r.DashPhase = tc.phase
				got := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(p.Iter(), emit) })

				// the reference draws each dash on its own, from the exact
				// arc lengths
				ref := &path.Data{}
				period := tc.dash[0] + tc.dash[1]
				for k := 0; float64(k)*period-tc.phase <= 2*turnAt+1e-6; k++ {
					s := float64(k)*period - tc.phase
					a, b := max(s, 0), min(s+tc.dash[0], 2*turnAt)
					if tc.dots {
						// a square of the line width, oriented along the
						// segment before any corner
						dir := vec.Vec2{X: 1}
						if a > turnAt+1e-6 {
							dir = dir2
						}
						c, h := at(o, a), width/2
						along, side := dir.Mul(h), dir.Rot90().Mul(h)
						ref.MoveTo(c.Sub(along).Sub(side))
						ref.LineTo(c.Add(along).Sub(side))
						ref.LineTo(c.Add(along).Add(side))
						ref.LineTo(c.Sub(along).Add(side))
						ref.Close()
						continue
					}
					if b-a < 1e-6 {
						continue // touches the path only at its end
					}
					ref.MoveTo(at(o, a))
					if a < turnAt-1e-6 && b > turnAt+1e-6 {
						ref.LineTo(at(o, turnAt))
					}
					ref.LineTo(at(o, b))
				}
				r = setup()
				var want [][]float32
				if tc.dots {
					want = renderGrid(t, r, func(emit func(int, int, []float32)) { r.FillNonZero(ref.Iter(), emit) })
				} else {
					want = renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(ref.Iter(), emit) })
				}

				compareGrids(t, want, got)
			})
		}
	}
}

// TestDashZeroLengthCases checks dashes which draw only their caps: a
// zero-length dash at the start point of a closed subpath, which takes the
// tangent of the closing segment, and a dash too short to stroke, which is
// drawn like a zero-length one.
func TestDashZeroLengthCases(t *testing.T) {
	const scale = 100
	setup := func(lineCap path.CapStyle, width float64) *Rasterizer {
		r := NewRasterizer(image.Rect(0, 0, 120, 120))
		r.CTM = matrix.Matrix{scale, 0, 0, scale, 10, 10}
		r.Width = width
		r.Cap = lineCap
		r.Join = path.JoinMiter
		r.MiterLimit = 10
		return r
	}

	t.Run("closed start point", func(t *testing.T) {
		// the closing segment runs along the diagonal (-0.8, -0.6)
		p := (&path.Data{}).MoveTo(vec.Vec2{X: 0.1, Y: 0.1})
		p.LineTo(vec.Vec2{X: 0.9, Y: 0.1})
		p.LineTo(vec.Vec2{X: 0.9, Y: 0.7})
		p.Close()

		r := setup(path.CapSquare, 0.1)
		r.Dash = []float64{0, 10}
		got := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(p.Iter(), emit) })

		c, h := vec.Vec2{X: 0.1, Y: 0.1}, 0.05
		along := vec.Vec2{X: -0.8, Y: -0.6}.Mul(h)
		side := along.Rot90()
		square := (&path.Data{}).MoveTo(c.Sub(along).Sub(side))
		square.LineTo(c.Add(along).Sub(side))
		square.LineTo(c.Add(along).Add(side))
		square.LineTo(c.Sub(along).Add(side))
		square.Close()
		r = setup(path.CapSquare, 0.1)
		want := renderGrid(t, r, func(emit func(int, int, []float32)) { r.FillNonZero(square.Iter(), emit) })

		compareGrids(t, want, got)
	})

	t.Run("dash too short to stroke", func(t *testing.T) {
		// a path 0.001 long, of which the first dash covers 5e-11
		const s = 1e-3
		r := NewRasterizer(image.Rect(0, 0, 40, 40))
		r.CTM = matrix.Matrix{1e4, 0, 0, 1e4, -1e4 + 20, -1e4 + 20}
		r.Width = s
		r.Cap = path.CapRound
		r.Dash = []float64{5e-11, 1}
		p := (&path.Data{}).MoveTo(vec.Vec2{X: 1, Y: 1}).LineTo(vec.Vec2{X: 1 + s, Y: 1})
		got := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(p.Iter(), emit) })

		r.Dash = nil
		dot := (&path.Data{}).MoveTo(vec.Vec2{X: 1, Y: 1}).LineTo(vec.Vec2{X: 1, Y: 1})
		want := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(dot.Iter(), emit) })

		compareGrids(t, want, got)
	})
}

// TestDashClosedMergeWithTrailingDot checks the merging of the last and the
// first dash of a closed subpath where a zero-length dash follows the last
// dash, at the start point.
func TestDashClosedMergeWithTrailingDot(t *testing.T) {
	// a 3-4-5 triangle with perimeter 2.4, corners at arc length 0.8 and 1.4
	v := []vec.Vec2{{X: 0.1, Y: 0.1}, {X: 0.9, Y: 0.1}, {X: 0.9, Y: 0.7}}
	at := func(s float64) vec.Vec2 {
		switch {
		case s <= 0.8:
			return v[0].Add(vec.Vec2{X: s})
		case s <= 1.4:
			return v[1].Add(vec.Vec2{Y: s - 0.8})
		default:
			return v[2].Add(vec.Vec2{X: -0.8, Y: -0.6}.Mul(s - 1.4))
		}
	}
	setup := func() *Rasterizer {
		r := NewRasterizer(image.Rect(0, 0, 120, 120))
		r.CTM = matrix.Matrix{100, 0, 0, 100, 10, 10}
		r.Width = 0.05
		r.Cap = path.CapRound
		r.Join = path.JoinRound
		return r
	}

	p := (&path.Data{}).MoveTo(v[0])
	p.LineTo(v[1])
	p.LineTo(v[2])
	p.Close()
	r := setup()
	// With phase 0.2 the dashes are [-0.2, 0.3], [0.5, 1.0], [1.2, 1.7]
	// and [1.9, 2.4], each followed by a zero-length dash.  The last dash
	// ends on the start point, followed by the zero-length dash there.
	r.Dash = []float64{0.5, 0, 0, 0.2}
	r.DashPhase = 0.2
	got := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(p.Iter(), emit) })

	ref := &path.Data{}
	ref.MoveTo(at(1.9)).LineTo(v[0]).LineTo(at(0.3))
	ref.MoveTo(at(0.5)).LineTo(v[1]).LineTo(at(1.0))
	ref.MoveTo(at(1.2)).LineTo(v[2]).LineTo(at(1.7))
	for _, s := range []float64{0.3, 1.0, 1.7} {
		ref.MoveTo(at(s)).LineTo(at(s))
	}
	ref.MoveTo(v[0]).LineTo(v[0])
	r = setup()
	want := renderGrid(t, r, func(emit func(int, int, []float32)) { r.Stroke(ref.Iter(), emit) })

	compareGrids(t, want, got)
}
