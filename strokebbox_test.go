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
	"math/rand"
	"testing"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
)

const (
	// bboxFlatness is the flattening tolerance used for the strokes.  Arcs
	// and folded joins make the drawn stroke fall short of the ideal one by
	// at most this much.
	bboxFlatness = 0.05

	// bboxNoise is the coverage below which a pixel counts as empty,
	// allowing for rounding in the float32 coverage accumulation.
	bboxNoise = 1e-4

	// bboxReachTol is how far short of a box edge the drawn stroke may stop.
	// Beyond the flatness allowance, the narrowest feature a stroke can reach
	// a box edge with is a miter tip, whose interior angle phi has
	// sin(phi/2) >= 1/MiterLimit.  With a miter limit of at most 10, a tip
	// reaching 0.1 pixel past a pixel boundary covers more than 5e-4 of a
	// pixel there, well above bboxNoise.
	bboxReachTol = bboxFlatness + 0.1

	// bboxMaxMiterLimit is the largest miter limit the test uses; see
	// bboxReachTol.
	bboxMaxMiterLimit = 10
)

// strokeBBoxCase is a set of polylines together with the stroke parameters
// they are drawn with.
type strokeBBoxCase struct {
	subpaths [][]vec.Vec2
	closed   bool
	opt      path.StrokeOptions

	// the dash pattern the rasterizer draws with, if opt.Dashed is set
	dash  []float64
	phase float64
}

func (c *strokeBBoxCase) String() string {
	return fmt.Sprintf("subpaths=%v closed=%v width=%g cap=%v join=%v miterLimit=%g dash=%v phase=%g",
		c.subpaths, c.closed, c.opt.Width, c.opt.Cap, c.opt.Join, c.opt.MiterLimit, c.dash, c.phase)
}

// TestPolylineStrokeBBox checks the boxes computed by
// [path.PolylineStrokeBBox] against the pixels the rasterizer covers when it
// strokes the same polylines under the identity CTM.
//
// The documented contract is that the box is the smallest one containing
// the stroke, and that there is none where the stroke draws nothing.  Both
// halves are checked on every side of the box: the stroke stays inside it,
// and reaches it.
//
// Pixels resolve a position only to within one pixel, so each check first
// shifts the polylines to put the box edge it tests on a pixel boundary.  A
// stroke crossing that edge then covers part of the first pixel beyond it,
// and a stroke reaching the edge covers part of the last pixel before it.
func TestPolylineStrokeBBox(t *testing.T) {
	const trials = 150
	for _, closed := range []bool{false, true} {
		for _, join := range []path.JoinStyle{path.JoinMiter, path.JoinRound, path.JoinBevel} {
			for _, lineCap := range []path.CapStyle{path.CapButt, path.CapRound, path.CapSquare} {
				name := fmt.Sprintf("closed=%v/%v/%v", closed, join, lineCap)
				t.Run(name, func(t *testing.T) {
					rng := rand.New(rand.NewSource(1))
					drawn := 0
					for range trials {
						c := randomStrokeBBoxCase(rng, closed, join, lineCap)
						if checkStrokeBBox(t, c) {
							drawn++
						}
						if t.Failed() {
							return
						}
					}

					// guard against a generator which leaves the checks
					// with nothing to test
					if drawn < trials*3/4 {
						t.Errorf("only %d of %d trials drew anything", drawn, trials)
					}
				})
			}
		}
	}
}

// TestPolylineStrokeBBoxZeroLength checks zero-length lines, which only round
// caps draw, as a disc of the line width.
//
// An open sub-path with a single vertex is left out: the polyline stands for
// a zero-length line, but the rasterizer is given a lone MoveTo, which draws
// nothing whatever the caps.
func TestPolylineStrokeBBoxZeroLength(t *testing.T) {
	p := vec.Vec2{X: 10.3, Y: 20.6}
	line := []vec.Vec2{{X: 0, Y: 0}, {X: 15, Y: 5}}
	for _, closed := range []bool{false, true} {
		shapes := [][][]vec.Vec2{
			{{p, p}},
			{{p, p, p}},
			{line, {p, p}},
		}
		if closed {
			shapes = append(shapes, [][]vec.Vec2{{p}})
		}
		for _, lineCap := range []path.CapStyle{path.CapButt, path.CapRound, path.CapSquare} {
			for _, subpaths := range shapes {
				c := &strokeBBoxCase{
					subpaths: subpaths,
					closed:   closed,
					opt: path.StrokeOptions{
						Width:      7,
						Cap:        lineCap,
						Join:       path.JoinMiter,
						MiterLimit: bboxMaxMiterLimit,
					},
				}
				drawn := checkStrokeBBox(t, c)
				if wantDrawn := lineCap == path.CapRound || len(subpaths) > 1; drawn != wantDrawn {
					t.Errorf("%v: drawn = %v, want %v", c, drawn, wantDrawn)
				}
			}
		}
	}
}

// TestPolylineStrokeBBoxDashed checks the boxes computed for dashed strokes,
// which have to hold the stroke whatever the dash pattern.
func TestPolylineStrokeBBoxDashed(t *testing.T) {
	const trials = 100
	for _, closed := range []bool{false, true} {
		for _, join := range []path.JoinStyle{path.JoinMiter, path.JoinRound, path.JoinBevel} {
			for _, lineCap := range []path.CapStyle{path.CapButt, path.CapRound, path.CapSquare} {
				name := fmt.Sprintf("closed=%v/%v/%v", closed, join, lineCap)
				t.Run(name, func(t *testing.T) {
					rng := rand.New(rand.NewSource(1))
					for range trials {
						c := randomStrokeBBoxCase(rng, closed, join, lineCap)
						c.opt.Dashed = true
						c.dash = make([]float64, 1+rng.Intn(4))
						for i := range c.dash {
							if rng.Float64() < 0.2 {
								continue // a zero-length dash or gap
							}
							c.dash[i] = c.opt.Width * math.Exp(math.Log(0.05)+rng.Float64()*math.Log(100))
						}
						c.dash[0] += c.opt.Width // not all zero
						c.phase = 10 * c.opt.Width * rng.Float64()

						bbox, ok := path.PolylineStrokeBBox(c.subpaths, c.closed, c.opt)
						if !ok {
							continue
						}
						checkStrokeBBoxContains(t, c, bbox)
						if t.Failed() {
							return
						}
					}
				})
			}
		}
	}

	// A dash ending on a bevelled sharp corner, with a square cap pointing
	// on along the first segment, reaches the box, and beyond the box of the
	// undashed stroke.
	pts := []vec.Vec2{{X: 100, Y: 30}, {X: 0, Y: 0}, {X: 100, Y: -30}}
	c := &strokeBBoxCase{
		subpaths: [][]vec.Vec2{pts},
		opt: path.StrokeOptions{
			Width: 8, Cap: path.CapSquare, Join: path.JoinBevel, Dashed: true,
		},
		dash: []float64{math.Hypot(100, 30), 1000},
	}
	bbox, _ := path.PolylineStrokeBBox(c.subpaths, c.closed, c.opt)
	if !checkStrokeBBoxContains(t, c, bbox) {
		t.Fatalf("%v: no pixels are covered", c)
	}
	const tol = bboxReachTol
	ext, b := strokeBBoxExtent(c, vec.Vec2{X: bbox.LLx + tol, Y: bbox.LLy + tol})
	if ext.Min.X >= gridLine(b.LLx+tol) {
		t.Errorf("%v: pixels %v stop short of the left edge of %v", c, ext, b)
	}
	c.opt.Dashed = false
	undashed, _ := path.PolylineStrokeBBox(c.subpaths, c.closed, c.opt)
	if undashed.LLx <= bbox.LLx+1 {
		t.Errorf("undashed bbox %v reaches as far as the dashed one %v", undashed, bbox)
	}
}

// randomStrokeBBoxCase returns one to three random walks with the given
// stroke style.  Step lengths range from a small fraction of the line width
// to several line widths, some steps repeat the previous vertex, and many
// run at multiples of 45 degrees, so that stroke edges line up with the
// box edges.  Some sub-paths are degenerate, with all vertices coinciding.
func randomStrokeBBoxCase(rng *rand.Rand, closed bool, join path.JoinStyle, lineCap path.CapStyle) *strokeBBoxCase {
	limits := []float64{1, 1.5, 2, 4, bboxMaxMiterLimit}
	c := &strokeBBoxCase{
		closed: closed,
		opt: path.StrokeOptions{
			Width:      4 + 26*rng.Float64(),
			Cap:        lineCap,
			Join:       join,
			MiterLimit: limits[rng.Intn(len(limits))],
		},
	}

	for range 1 + rng.Intn(3) {
		p := vec.Vec2{X: 100 * rng.Float64(), Y: 100 * rng.Float64()}
		pts := []vec.Vec2{p}
		degenerate := rng.Float64() < 0.1
		for range 1 + rng.Intn(5) {
			if !degenerate && rng.Float64() >= 0.1 {
				length := c.opt.Width * math.Exp(math.Log(0.05)+rng.Float64()*math.Log(120))
				angle := 2 * math.Pi * rng.Float64()
				if rng.Float64() < 0.3 {
					angle = float64(rng.Intn(8)) * math.Pi / 4
				}
				p = p.Add(vec.Vec2{X: math.Cos(angle), Y: math.Sin(angle)}.Mul(length))
			}
			pts = append(pts, p)
		}
		c.subpaths = append(c.subpaths, pts)
	}
	return c
}

// checkStrokeBBox checks the box for c on all four sides, and reports
// whether the stroke covers any pixels.
func checkStrokeBBox(t *testing.T, c *strokeBBoxCase) (drawn bool) {
	t.Helper()

	bbox, ok := path.PolylineStrokeBBox(c.subpaths, c.closed, c.opt)
	if !ok {
		if ext, _ := strokeBBoxExtent(c, vec.Vec2{}); !ext.Empty() {
			t.Errorf("%v: no bounding box, but pixels %v are covered", c, ext)
		}
		return false
	}

	if !checkStrokeBBoxContains(t, c, bbox) {
		t.Errorf("%v: bounding box %v, but no pixels are covered", c, bbox)
		return false
	}

	// the stroke reaches the box
	const tol = bboxReachTol
	hi := vec.Vec2{X: bbox.URx, Y: bbox.URy}
	lo := vec.Vec2{X: bbox.LLx, Y: bbox.LLy}
	ext, b := strokeBBoxExtent(c, hi.Sub(vec.Vec2{X: tol, Y: tol}))
	if ext.Max.X <= gridLine(b.URx-tol) || ext.Max.Y <= gridLine(b.URy-tol) {
		t.Errorf("%v: pixels %v stop short of the upper edges of %v", c, ext, b)
	}
	ext, b = strokeBBoxExtent(c, lo.Add(vec.Vec2{X: tol, Y: tol}))
	if ext.Min.X >= gridLine(b.LLx+tol) || ext.Min.Y >= gridLine(b.LLy+tol) {
		t.Errorf("%v: pixels %v stop short of the lower edges of %v", c, ext, b)
	}

	return true
}

// checkStrokeBBoxContains checks that the stroke of c stays within bbox, the
// box of its unshifted polylines, and reports whether it covers any pixels.
func checkStrokeBBoxContains(t *testing.T, c *strokeBBoxCase, bbox rect.Rect) (drawn bool) {
	t.Helper()

	ext, b := strokeBBoxExtent(c, vec.Vec2{X: bbox.URx, Y: bbox.URy})
	if ext.Empty() {
		return false
	}
	if ext.Max.X > gridLine(b.URx) || ext.Max.Y > gridLine(b.URy) {
		t.Errorf("%v: pixels %v reach beyond the upper edges of %v", c, ext, b)
	}
	ext, b = strokeBBoxExtent(c, vec.Vec2{X: bbox.LLx, Y: bbox.LLy})
	if ext.Min.X < gridLine(b.LLx) || ext.Min.Y < gridLine(b.LLy) {
		t.Errorf("%v: pixels %v reach beyond the lower edges of %v", c, ext, b)
	}
	return true
}

// strokeBBoxExtent shifts the polylines of c so that the point target lands
// on a pixel corner, strokes them, and returns the smallest rectangle of
// pixels holding all covered pixels, together with the box of the shifted
// polylines.
func strokeBBoxExtent(c *strokeBBoxCase, target vec.Vec2) (image.Rectangle, rect.Rect) {
	// A bound on the stroke which does not rely on the box under test: no
	// part of it lies further from a vertex than a miter tip or the corner
	// of a square cap.
	reach := c.opt.Width/2*max(c.opt.MiterLimit, math.Sqrt2) + 1
	var vertices rect.Rect
	for i, pts := range c.subpaths {
		for j, p := range pts {
			if i == 0 && j == 0 {
				vertices = rect.Rect{LLx: p.X, LLy: p.Y, URx: p.X, URy: p.Y}
			} else {
				vertices.Add(p.X, p.Y)
			}
		}
	}
	const margin = 4
	base := vec.Vec2{X: margin - vertices.LLx + reach, Y: margin - vertices.LLy + reach}
	shift := vec.Vec2{
		X: math.Ceil(target.X+base.X) - target.X,
		Y: math.Ceil(target.Y+base.Y) - target.Y,
	}
	clip := image.Rect(0, 0,
		int(math.Ceil(vertices.Dx()+2*reach))+2*margin+1,
		int(math.Ceil(vertices.Dy()+2*reach))+2*margin+1)

	shifted := make([][]vec.Vec2, len(c.subpaths))
	p := &path.Data{}
	for i, pts := range c.subpaths {
		shifted[i] = make([]vec.Vec2, len(pts))
		for j, q := range pts {
			shifted[i][j] = q.Add(shift)
		}
		p.MoveTo(shifted[i][0])
		for _, q := range shifted[i][1:] {
			p.LineTo(q)
		}
		if c.closed {
			p.Close()
		}
	}
	bbox, _ := path.PolylineStrokeBBox(shifted, c.closed, c.opt)

	r := NewRasterizer(clip)
	r.Flatness = bboxFlatness
	r.Width = c.opt.Width
	r.Cap = c.opt.Cap
	r.Join = c.opt.Join
	r.MiterLimit = c.opt.MiterLimit
	if c.opt.Dashed {
		r.Dash = c.dash
		r.DashPhase = c.phase
	}

	var ext image.Rectangle
	r.Stroke(p.Iter(), func(y, xMin int, coverage []float32) {
		for k, cov := range coverage {
			if cov > bboxNoise {
				ext = ext.Union(image.Rect(xMin+k, y, xMin+k+1, y+1))
			}
		}
	})
	return ext, bbox
}

// gridLine returns the pixel boundary x, which has been placed within
// rounding error of one.
func gridLine(x float64) int {
	return int(math.Round(x))
}
