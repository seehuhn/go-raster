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
	"testing"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
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
		r.Cap = graphics.LineCapSquare
		r.Join = graphics.LineJoinRound
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
