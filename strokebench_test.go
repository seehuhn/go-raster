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

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// sink prevents the emit callback from being optimised away.
var sink float32

// emitSink is an emit callback that accumulates coverage into sink.
func emitSink(y, xMin int, coverage []float32) {
	for _, c := range coverage {
		sink += c
	}
}

// makeWavePath builds a curvy path from cubic Béziers (n arches of a wave).
func makeWavePath(n int, w, h float64) *path.Data {
	p := &path.Data{}
	dx := w / float64(n)
	p.MoveTo(vec.Vec2{X: 0, Y: h / 2})
	for i := range n {
		x0 := float64(i) * dx
		s := h * 0.4
		if i%2 == 1 {
			s = -s
		}
		p.CubeTo(
			vec.Vec2{X: x0 + dx/3, Y: h/2 + s},
			vec.Vec2{X: x0 + 2*dx/3, Y: h/2 - s},
			vec.Vec2{X: x0 + dx, Y: h / 2},
		)
	}
	return p
}

// makeZigZag builds a polyline with n corners.
func makeZigZag(n int, w, h float64) *path.Data {
	p := &path.Data{}
	dx := w / float64(n)
	p.MoveTo(vec.Vec2{X: 0, Y: h * 0.3})
	for i := 1; i <= n; i++ {
		y := h * 0.3
		if i%2 == 1 {
			y = h * 0.7
		}
		p.LineTo(vec.Vec2{X: float64(i) * dx, Y: y})
	}
	return p
}

// makeSpiral builds a long spiral polyline (lots of small segments).
func makeSpiral(turns int, cx, cy float64) *path.Data {
	p := &path.Data{}
	p.MoveTo(vec.Vec2{X: cx, Y: cy})
	steps := turns * 60
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		ang := t * float64(turns) * 2 * math.Pi
		r := t * min(cx, cy) * 0.9
		p.LineTo(vec.Vec2{X: cx + r*math.Cos(ang), Y: cy + r*math.Sin(ang)})
	}
	return p
}

// benchStroke runs r.Stroke on p, applying setup to configure the
// rasterizer before timing starts.
func benchStroke(b *testing.B, p *path.Data, setup func(r *Rasterizer)) {
	const size = 500
	clip := rect.Rect{LLx: 0, LLy: 0, URx: size, URy: size}
	r := NewRasterizer(clip)
	r.Width = 4
	if setup != nil {
		setup(r)
	}
	iter := p.Iter()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		r.Stroke(iter, emitSink)
	}
}

// BenchmarkStroke covers curve-heavy, corner-heavy, thick, and dashed
// strokes with miter and round joins.
func BenchmarkStroke(b *testing.B) {
	wave := makeWavePath(8, 500, 500)
	zig := makeZigZag(60, 500, 500)
	spiral := makeSpiral(12, 250, 250)

	cases := []struct {
		name  string
		path  *path.Data
		setup func(r *Rasterizer)
	}{
		{"wave/round", wave, func(r *Rasterizer) {
			r.Cap = graphics.LineCapRound
			r.Join = graphics.LineJoinRound
		}},
		{"wave/miter", wave, nil},
		{"wave/thick", wave, func(r *Rasterizer) { r.Width = 20 }},
		{"zigzag/miter", zig, nil},
		{"zigzag/round", zig, func(r *Rasterizer) {
			r.Cap = graphics.LineCapRound
			r.Join = graphics.LineJoinRound
		}},
		{"spiral/round", spiral, func(r *Rasterizer) {
			r.Cap = graphics.LineCapRound
			r.Join = graphics.LineJoinRound
		}},
		{"wave/dashed", wave, func(r *Rasterizer) {
			r.Dash = []float64{8, 4}
			r.Cap = graphics.LineCapRound
		}},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) { benchStroke(b, c.path, c.setup) })
	}
}
