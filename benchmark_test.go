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
	"image/color"
	"testing"

	"golang.org/x/image/vector"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// BenchmarkRasterizerMethodA benchmarks using fillSmallPath (2D buffers).
func BenchmarkRasterizerMethodA(b *testing.B) {
	sizes := []int{20, 200, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			clip := image.Rect(0, 0, size, size)
			r := NewRasterizer(clip)
			r.smallPathThreshold = 1 << 30 // Force method A

			dst := image.NewAlpha(image.Rect(0, 0, size, size))

			center := float64(size) / 2
			outerR := float64(size) * 0.45
			innerR := float64(size) * 0.30

			oPath := makeOPath(center, center, outerR, innerR)
			oPathIter := oPath.Iter()

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				r.FillEvenOdd(oPathIter, func(y, xMin int, coverage []float32) {
					row := dst.Pix[y*dst.Stride+xMin:]
					for i, c := range coverage {
						row[i] = uint8(c * 255)
					}
				})
			}
		})
	}
}

// BenchmarkRasterizerMethodB benchmarks using fillLargePath (active edge list).
func BenchmarkRasterizerMethodB(b *testing.B) {
	sizes := []int{20, 200, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			clip := image.Rect(0, 0, size, size)
			r := NewRasterizer(clip)
			r.smallPathThreshold = 0 // Force method B

			dst := image.NewAlpha(image.Rect(0, 0, size, size))

			center := float64(size) / 2
			outerR := float64(size) * 0.45
			innerR := float64(size) * 0.30

			oPath := makeOPath(center, center, outerR, innerR)
			oPathIter := oPath.Iter()

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				r.FillEvenOdd(oPathIter, func(y, xMin int, coverage []float32) {
					row := dst.Pix[y*dst.Stride+xMin:]
					for i, c := range coverage {
						row[i] = uint8(c * 255)
					}
				})
			}
		})
	}
}

// BenchmarkVectorO benchmarks x/image/vector drawing an "O" shape.
func BenchmarkVectorO(b *testing.B) {
	sizes := []int{20, 200, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			r := vector.NewRasterizer(size, size)

			dst := image.NewAlpha(image.Rect(0, 0, size, size))
			src := image.NewUniform(color.Alpha{255})

			center := float32(size) / 2
			outerR := float32(size) * 0.45
			innerR := float32(size) * 0.30

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				r.Reset(size, size)

				// Outer circle (counter-clockwise)
				addCircleToVector(r, center, center, outerR, false)
				// Inner circle (clockwise)
				addCircleToVector(r, center, center, innerR, true)

				// Rasterize and composite
				r.Draw(dst, dst.Bounds(), src, image.Point{})
			}
		})
	}
}

// makeOPath creates an "O" shape path for our rasterizer.
// Outer circle is counter-clockwise, inner circle is clockwise.
func makeOPath(cx, cy, outerR, innerR float64) *path.Data {
	p := &path.Data{}
	// Outer circle (counter-clockwise)
	addCircleToData(p, cx, cy, outerR, false)
	// Inner circle (clockwise)
	addCircleToData(p, cx, cy, innerR, true)
	return p
}

// addCircleToData adds a circle to a path.Data using cubic Bézier curves.
func addCircleToData(p *path.Data, cx, cy, r float64, clockwise bool) {
	// Magic number for circular arc approximation with cubic Bézier
	const k = 0.5522847498
	kr := k * r

	pt := func(x, y float64) vec.Vec2 { return vec.Vec2{X: x, Y: y} }

	if clockwise {
		// Start at top, go clockwise
		p.MoveTo(pt(cx, cy-r))
		p.CubeTo(pt(cx-kr, cy-r), pt(cx-r, cy-kr), pt(cx-r, cy))
		p.CubeTo(pt(cx-r, cy+kr), pt(cx-kr, cy+r), pt(cx, cy+r))
		p.CubeTo(pt(cx+kr, cy+r), pt(cx+r, cy+kr), pt(cx+r, cy))
		p.CubeTo(pt(cx+r, cy-kr), pt(cx+kr, cy-r), pt(cx, cy-r))
	} else {
		// Start at top, go counter-clockwise
		p.MoveTo(pt(cx, cy-r))
		p.CubeTo(pt(cx+kr, cy-r), pt(cx+r, cy-kr), pt(cx+r, cy))
		p.CubeTo(pt(cx+r, cy+kr), pt(cx+kr, cy+r), pt(cx, cy+r))
		p.CubeTo(pt(cx-kr, cy+r), pt(cx-r, cy+kr), pt(cx-r, cy))
		p.CubeTo(pt(cx-r, cy-kr), pt(cx-kr, cy-r), pt(cx, cy-r))
	}
	p.Close()
}

// addCircleToVector adds a circle to a vector.Rasterizer using cubic Bézier curves.
func addCircleToVector(r *vector.Rasterizer, cx, cy, radius float32, clockwise bool) {
	const k = float32(0.5522847498)
	kr := k * radius

	if clockwise {
		r.MoveTo(cx, cy-radius)
		r.CubeTo(cx-kr, cy-radius, cx-radius, cy-kr, cx-radius, cy)
		r.CubeTo(cx-radius, cy+kr, cx-kr, cy+radius, cx, cy+radius)
		r.CubeTo(cx+kr, cy+radius, cx+radius, cy+kr, cx+radius, cy)
		r.CubeTo(cx+radius, cy-kr, cx+kr, cy-radius, cx, cy-radius)
	} else {
		r.MoveTo(cx, cy-radius)
		r.CubeTo(cx+kr, cy-radius, cx+radius, cy-kr, cx+radius, cy)
		r.CubeTo(cx+radius, cy+kr, cx+kr, cy+radius, cx, cy+radius)
		r.CubeTo(cx-kr, cy+radius, cx-radius, cy+kr, cx-radius, cy)
		r.CubeTo(cx-radius, cy-kr, cx-kr, cy-radius, cx, cy-radius)
	}
	r.ClosePath()
}

// BenchmarkSparseRows fills a thin diagonal across a wide clip using the
// active edge list: the bounding box spans the whole clip, but each row
// touches only a few pixels, so per-row work should scale with the touched
// span rather than the clip width.
func BenchmarkSparseRows(b *testing.B) {
	const width, height = 2000, 1000
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: 0, Y: 0}).
		LineTo(vec.Vec2{X: 3.3, Y: 0}).
		LineTo(vec.Vec2{X: width, Y: height}).
		LineTo(vec.Vec2{X: width - 3.3, Y: height}).
		Close()
	r := NewRasterizer(image.Rect(0, 0, width, height))
	r.smallPathThreshold = 0
	emit := func(y, xMin int, coverage []float32) {}

	b.ResetTimer()
	for b.Loop() {
		r.FillNonZero(p.Iter(), emit)
	}
}
