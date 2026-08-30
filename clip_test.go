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

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// thresholds selects the small-path (2D buffer) and large-path (active edge
// list) approaches in turn.
var thresholds = map[string]int{
	"small": 1 << 30,
	"large": 0,
}

// renderGrid draws into a grid the size of r.Clip and returns the coverage
// indexed as grid[y-Clip.Min.Y][x-Clip.Min.X].
func renderGrid(t *testing.T, r *Rasterizer, draw func(emit func(y, xMin int, cov []float32))) [][]float32 {
	t.Helper()
	grid := make([][]float32, r.Clip.Dy())
	for i := range grid {
		grid[i] = make([]float32, r.Clip.Dx())
	}
	draw(func(y, xMin int, cov []float32) {
		if y < r.Clip.Min.Y || y >= r.Clip.Max.Y {
			t.Fatalf("row %d outside clip %v", y, r.Clip)
		}
		if xMin < r.Clip.Min.X || xMin+len(cov) > r.Clip.Max.X {
			t.Fatalf("row %d spans [%d,%d), outside clip %v", y, xMin, xMin+len(cov), r.Clip)
		}
		copy(grid[y-r.Clip.Min.Y][xMin-r.Clip.Min.X:], cov)
	})
	return grid
}

// TestEdgesClippedToDevice checks that the edge list never extends beyond
// the clip: edges are trimmed to the clip's vertical extent, parts to the
// right of the clip are dropped, and parts to the left are replaced by a
// vertical edge one pixel left of the clip.  This bounds the work of every
// later stage by the clip size, whatever the input coordinates.
func TestEdgesClippedToDevice(t *testing.T) {
	clip := image.Rect(10, 20, 110, 120)
	r := NewRasterizer(clip)

	// a shape with edges crossing every clip boundary and edges entirely
	// above, below, left and right of the clip
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: -1e9, Y: 50}).
		LineTo(vec.Vec2{X: 1e9, Y: 50.5}).  // nearly horizontal, huge extent
		LineTo(vec.Vec2{X: 1e9, Y: 1e9}).   // entirely right of the clip
		LineTo(vec.Vec2{X: 60, Y: 1e9}).    // entirely below the clip
		LineTo(vec.Vec2{X: 60, Y: -1e9}).   // crosses the clip vertically
		LineTo(vec.Vec2{X: -1e9, Y: -1e9}). // entirely above the clip
		Close()                             // entirely left of the clip

	_, _, _, _, ok := r.collectPathEdges(p.Iter())
	if !ok {
		t.Fatal("expected edges")
	}
	xLo, xHi := float64(clip.Min.X-1), float64(clip.Max.X)
	yLo, yHi := float64(clip.Min.Y), float64(clip.Max.Y)
	for i, e := range r.edges {
		xOther := e.x + e.dxdy*(e.ymax-e.ymin)
		for _, x := range []float64{e.x, xOther} {
			if x < xLo || x > xHi {
				t.Errorf("edge %d: x=%g outside [%g,%g]", i, x, xLo, xHi)
			}
		}
		for _, y := range []float64{e.ymin, e.ymax} {
			if y < yLo || y > yHi {
				t.Errorf("edge %d: y=%g outside [%g,%g]", i, y, yLo, yHi)
			}
		}
	}
}

// TestFillExtremeVertex checks that a vertex far beyond the int range does
// not lose the edges leading to it, in either approach.
func TestFillExtremeVertex(t *testing.T) {
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: 0, Y: 0}).
		LineTo(vec.Vec2{X: 50, Y: 1e19}).
		LineTo(vec.Vec2{X: 100, Y: 0}).
		Close()
	for name, threshold := range thresholds {
		t.Run(name, func(t *testing.T) {
			r := NewRasterizer(image.Rect(0, 0, 100, 100))
			r.smallPathThreshold = threshold
			grid := renderGrid(t, r, func(emit func(int, int, []float32)) {
				r.FillNonZero(p.Iter(), emit)
			})
			for y, row := range grid {
				for x, c := range row {
					if c < 0.999 {
						t.Fatalf("pixel (%d,%d): coverage %g, want 1", x, y, c)
					}
				}
			}
		})
	}
}

// TestFillClipOffsetExtremeEdge checks a clip whose origin is not (0,0)
// against edges starting far above it: the edge must be active on every
// row of the clip, not just the last.
func TestFillClipOffsetExtremeEdge(t *testing.T) {
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: 0, Y: -1e19}).
		LineTo(vec.Vec2{X: 0, Y: 1e19}).
		LineTo(vec.Vec2{X: 1000, Y: 1e19}).
		LineTo(vec.Vec2{X: 1000, Y: -1e19}).
		Close()
	for name, threshold := range thresholds {
		t.Run(name, func(t *testing.T) {
			r := NewRasterizer(image.Rect(0, 1, 1000, 1000))
			r.smallPathThreshold = threshold
			grid := renderGrid(t, r, func(emit func(int, int, []float32)) {
				r.FillNonZero(p.Iter(), emit)
			})
			for y, row := range grid {
				for x, c := range row {
					if c < 0.999 {
						t.Fatalf("pixel (%d,%d): coverage %g, want 1", x, y+r.Clip.Min.Y, c)
					}
				}
			}
		})
	}
}

// TestClipConsistency checks that rendering into a sub-rectangle of the
// clip gives exactly the corresponding part of the full rendering, for
// fills under both rules and for a stroke.  This is the oracle for edge
// clipping: replacing the parts of an edge outside the clip must not
// change the winding seen by any pixel inside.
func TestClipConsistency(t *testing.T) {
	// a self-overlapping star with curves, spanning the full clip
	star := &path.Data{}
	const cx, cy, rad = 100, 100, 130
	for i := range 5 {
		a := float64(i) * 4 * math.Pi / 5
		pt := vec.Vec2{X: cx + rad*math.Cos(a), Y: cy + rad*math.Sin(a)}
		if i == 0 {
			star.MoveTo(pt)
		} else {
			ctrl := vec.Vec2{X: cx + 0.3*rad*math.Cos(a-1), Y: cy + 0.3*rad*math.Sin(a-1)}
			star.QuadTo(ctrl, pt)
		}
	}
	star.Close()

	full := image.Rect(0, 0, 200, 200)
	sub := image.Rect(37, 51, 143, 160)
	ops := map[string]func(r *Rasterizer, emit func(int, int, []float32)){
		"nonzero": func(r *Rasterizer, emit func(int, int, []float32)) { r.FillNonZero(star.Iter(), emit) },
		"evenodd": func(r *Rasterizer, emit func(int, int, []float32)) { r.FillEvenOdd(star.Iter(), emit) },
		"stroke": func(r *Rasterizer, emit func(int, int, []float32)) {
			r.Width = 9
			r.Dash = []float64{40, 7}
			r.Stroke(star.Iter(), emit)
		},
	}
	for opName, op := range ops {
		for thName, threshold := range thresholds {
			t.Run(opName+"/"+thName, func(t *testing.T) {
				rFull := NewRasterizer(full)
				rFull.smallPathThreshold = threshold
				rFull.CTM = matrix.Matrix{0.9, 0.2, -0.3, 1.1, 5, -3}
				want := renderGrid(t, rFull, func(emit func(int, int, []float32)) { op(rFull, emit) })

				rSub := NewRasterizer(sub)
				rSub.smallPathThreshold = threshold
				rSub.CTM = rFull.CTM
				got := renderGrid(t, rSub, func(emit func(int, int, []float32)) { op(rSub, emit) })

				for y := range sub.Dy() {
					for x := range sub.Dx() {
						w := want[y+sub.Min.Y-full.Min.Y][x+sub.Min.X-full.Min.X]
						g := got[y][x]
						if math.Abs(float64(w-g)) > 1e-5 {
							t.Fatalf("pixel (%d,%d): sub-clip coverage %g, full-clip %g",
								x+sub.Min.X, y+sub.Min.Y, g, w)
						}
					}
				}
			})
		}
	}
}

// TestClipOverflowNoNaN checks that an edge whose device-space x span
// overflows float64 relative to its vertical extent produces no non-finite
// edges and no non-finite coverage.  Such an edge makes dx/dy infinite, and
// the clip splits in addEdge would otherwise divide one infinity by another.
func TestClipOverflowNoNaN(t *testing.T) {
	// The CTM scales x far enough that the two endpoints below straddle the
	// float64 range, so their difference overflows while both stay finite.
	ctm := matrix.Matrix{1e300, 0, 0, 1, 0, 0}
	p := (&path.Data{}).
		MoveTo(vec.Vec2{X: -1e8, Y: -1}).
		LineTo(vec.Vec2{X: 1e8, Y: 200}).
		LineTo(vec.Vec2{X: 5, Y: 50}).
		Close()

	ops := map[string]func(r *Rasterizer, emit func(int, int, []float32)){
		"nonzero": func(r *Rasterizer, emit func(int, int, []float32)) { r.FillNonZero(p.Iter(), emit) },
		"evenodd": func(r *Rasterizer, emit func(int, int, []float32)) { r.FillEvenOdd(p.Iter(), emit) },
		"stroke": func(r *Rasterizer, emit func(int, int, []float32)) {
			r.Width = 3
			r.Stroke(p.Iter(), emit)
		},
	}
	for opName, op := range ops {
		for thName, threshold := range thresholds {
			t.Run(opName+"/"+thName, func(t *testing.T) {
				r := NewRasterizer(image.Rect(0, 0, 100, 100))
				r.smallPathThreshold = threshold
				r.CTM = ctm
				op(r, func(y, xMin int, cov []float32) {
					for i, c := range cov {
						if math.IsNaN(float64(c)) || math.IsInf(float64(c), 0) {
							t.Fatalf("coverage %v at (%d,%d)", c, xMin+i, y)
						}
					}
				})
				for i, e := range r.edges {
					if math.IsNaN(e.ymin) || math.IsNaN(e.ymax) ||
						math.IsNaN(e.x) || math.IsNaN(e.dxdy) {
						t.Fatalf("edge %d is not finite: %+v", i, e)
					}
				}
			})
		}
	}
}
