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
	"testing"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// mustPanic runs fn and fails the test unless fn panics.
func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic, got none", name)
		}
	}()
	fn()
}

// A non-positive Flatness would make the curve-flattening segment count
// overflow to a huge value, hanging the rasteriser. Both fill and stroke
// must reject it up front.
func TestPanicOnInvalidFlatness(t *testing.T) {
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: 0, Y: 0}).
		LineTo(vec.Vec2{X: 10, Y: 0}).
		LineTo(vec.Vec2{X: 10, Y: 10}).
		Close()
	clip := rect.Rect{LLx: 0, LLy: 0, URx: 20, URy: 20}
	emit := func(y, xMin int, coverage []float32) {}

	for _, flatness := range []float64{0, -1} {
		mustPanic(t, "FillNonZero", func() {
			r := NewRasterizer(clip)
			r.Flatness = flatness
			r.FillNonZero(p.Iter(), emit)
		})
		mustPanic(t, "Stroke", func() {
			r := NewRasterizer(clip)
			r.Flatness = flatness
			r.Stroke(p.Iter(), emit)
		})
	}
}

// Stroke requires a positive Width.
func TestPanicOnInvalidWidth(t *testing.T) {
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: 0, Y: 0}).
		LineTo(vec.Vec2{X: 10, Y: 10})
	clip := rect.Rect{LLx: 0, LLy: 0, URx: 20, URy: 20}
	emit := func(y, xMin int, coverage []float32) {}

	for _, width := range []float64{0, -1} {
		mustPanic(t, "Stroke", func() {
			r := NewRasterizer(clip)
			r.Width = width
			r.Stroke(p.Iter(), emit)
		})
	}
}

// buildFuzzPath turns arbitrary bytes into a path. Drawing commands before a
// MoveTo (or after a Close) are dropped, matching path-builder semantics.
func buildFuzzPath(data []byte) *path.Data {
	p := &path.Data{}
	i := 0
	readF := func() float64 {
		if i >= len(data) {
			return 0
		}
		b := data[i]
		i++
		return float64(b)/255*300 - 50 // [-50, 250]
	}
	readPt := func() vec.Vec2 { return vec.Vec2{X: readF(), Y: readF()} }

	started := false
	for i < len(data) {
		cmd := data[i] % 5
		i++
		switch cmd {
		case 0:
			p.MoveTo(readPt())
			started = true
		case 1:
			if started {
				p.LineTo(readPt())
			}
		case 2:
			if started {
				p.QuadTo(readPt(), readPt())
			}
		case 3:
			if started {
				p.CubeTo(readPt(), readPt(), readPt())
			}
		case 4:
			if started {
				p.Close()
				started = false
			}
		}
	}
	return p
}

// FuzzRasterize feeds arbitrary paths to every operation with valid parameters
// and checks that the rasteriser neither panics nor produces out-of-range
// coverage. Coordinates are bounded so degenerate inputs cannot legitimately
// blow up the segment count, distinguishing a genuine hang from slow input.
func FuzzRasterize(f *testing.F) {
	f.Add([]byte{0, 10, 10, 1, 200, 10, 1, 200, 200, 4})
	f.Add([]byte{0, 50, 50, 3, 100, 0, 0, 100, 200, 200})
	f.Add([]byte{0, 30, 30, 1, 30, 30, 2, 100, 100, 200, 30})

	clip := rect.Rect{LLx: 0, LLy: 0, URx: 200, URy: 200}

	f.Fuzz(func(t *testing.T, data []byte) {
		p := buildFuzzPath(data)
		iter := p.Iter()

		emit := func(y, xMin int, coverage []float32) {
			for _, c := range coverage {
				if c < 0 || c > 1 {
					t.Fatalf("coverage out of range: %v", c)
				}
			}
		}

		r := NewRasterizer(clip)
		r.FillNonZero(iter, emit)
		r.FillEvenOdd(iter, emit)

		// stroke with a few parameter combinations derived from the input
		var width float64 = 1
		var capStyle graphics.LineCapStyle
		var joinStyle graphics.LineJoinStyle
		var dash []float64
		var phase float64
		if len(data) > 0 {
			width = float64(data[0])/255*20 + 0.1 // [0.1, 20.1]
			capStyle = graphics.LineCapStyle(data[0] % 3)
			joinStyle = graphics.LineJoinStyle(data[0] % 3)
			if data[0]%2 == 1 {
				dash = []float64{5, 3}
				phase = float64(data[0])
			}
		}
		r.Width = width
		r.Cap = capStyle
		r.Join = joinStyle
		r.Dash = dash
		r.DashPhase = phase
		r.Stroke(iter, emit)
	})
}
