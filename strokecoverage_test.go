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
	"math"
	"math/rand"
	"testing"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// distToSegment returns the distance from p to the segment ab.
func distToSegment(p, a, b vec.Vec2) float64 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 == 0 {
		return p.Sub(a).Length()
	}
	t := max(0, min(1, p.Sub(a).Dot(ab)/l2))
	return p.Sub(a.Add(ab.Mul(t))).Length()
}

// checkStrokeCoverage strokes the polyline pts with round caps and round
// joins and compares the result against the exact stroked region.
//
// With round caps and joins the stroke of a polyline is exactly the set of
// points within half a line width of it, so the distance to the path decides
// whether a pixel must be covered.  Pixels near the boundary are skipped,
// since there partial coverage is expected.  The check catches both a stroke
// that spills outside its region and one with holes punched out of it.
func checkStrokeCoverage(t *testing.T, label string, pts []vec.Vec2, closed bool, width float64) {
	t.Helper()
	const size = 100

	p := (&path.Data{}).MoveTo(pts[0])
	for _, q := range pts[1:] {
		p.LineTo(q)
	}
	if closed {
		p.Close()
	}

	r := NewRasterizer(rect.Rect{URx: size, URy: size})
	r.Width = width
	r.Join = graphics.LineJoinRound
	r.Cap = graphics.LineCapRound
	r.MiterLimit = 10

	covered := make([]bool, size*size)
	r.Stroke(p.Iter(), func(y, xMin int, coverage []float32) {
		for k, c := range coverage {
			covered[y*size+xMin+k] = c > 0.5
		}
	})

	d := width / 2
	numSeg := len(pts) - 1
	if closed {
		numSeg = len(pts)
	}
	for y := range size {
		for x := range size {
			c := vec.Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			dist := math.Inf(1)
			for k := range numSeg {
				dist = min(dist, distToSegment(c, pts[k], pts[(k+1)%len(pts)]))
			}
			if math.Abs(dist-d) < 1 {
				continue // antialiased boundary
			}
			if want := dist < d; covered[y*size+x] != want {
				t.Fatalf("%s: pixel (%d,%d) covered=%v, want %v (distance %.3f, half width %.3f, %d points, closed=%v)",
					label, x, y, covered[y*size+x], want, dist, d, len(pts), closed)
			}
		}
	}
}

// TestStrokeCoverageAgainstDistanceField checks strokes of random polylines
// with few, mostly sharp corners against the exact stroked region.
func TestStrokeCoverageAgainstDistanceField(t *testing.T) {
	for _, seed := range []int64{1, 7, 42} {
		rng := rand.New(rand.NewSource(seed))
		for i := range 70 {
			numPts := 2 + rng.Intn(5)
			pts := make([]vec.Vec2, numPts)
			for j := range pts {
				pts[j] = vec.Vec2{
					X: 25 + rng.Float64()*50,
					Y: 25 + rng.Float64()*50,
				}
			}
			closed := i%2 == 0
			width := 2 + rng.Float64()*16

			label := fmt.Sprintf("seed %d case %d", seed, i)
			checkStrokeCoverage(t, label, pts, closed, width)
		}
	}
}

// TestStrokeCoverageSmoothPaths checks strokes of dense, gently turning
// polylines -- the shape curve flattening produces -- against the exact
// stroked region.  The open paths are random smooth walks; the closed paths
// are perturbed circles, half of them with a single sharp vertex.
func TestStrokeCoverageSmoothPaths(t *testing.T) {
	centre := vec.Vec2{X: 50, Y: 50}

	for _, seed := range []int64{1, 7, 42} {
		rng := rand.New(rand.NewSource(seed))

		for i := range 12 {
			pts := smoothWalk(rng)
			width := 2 + rng.Float64()*22
			label := fmt.Sprintf("open seed %d case %d", seed, i)
			checkStrokeCoverage(t, label, pts, false, width)
		}

		for i := range 12 {
			numPts := 40 + rng.Intn(40)
			radius := 12 + rng.Float64()*10
			m := float64(2 + rng.Intn(3))
			phase := rng.Float64() * 2 * math.Pi
			amp := rng.Float64() * 0.2
			pts := make([]vec.Vec2, numPts)
			for j := range pts {
				u := 2 * math.Pi * float64(j) / float64(numPts)
				rj := radius * (1 + amp*math.Sin(m*u+phase))
				pts[j] = centre.Add(vec.Vec2{X: rj * math.Cos(u), Y: rj * math.Sin(u)})
			}
			if i%2 == 0 {
				// one sharp corner, away from the seam
				k := numPts / 2
				pts[k] = centre.Add(pts[k].Sub(centre).Mul(1.6))
			}
			width := 2 + rng.Float64()*22
			label := fmt.Sprintf("closed seed %d case %d", seed, i)
			checkStrokeCoverage(t, label, pts, true, width)
		}
	}
}

// smoothWalk builds a dense polyline with small random turns, steering back
// towards the canvas centre when it drifts too far out.
func smoothWalk(rng *rand.Rand) []vec.Vec2 {
	centre := vec.Vec2{X: 50, Y: 50}
	pos := vec.Vec2{X: 35 + rng.Float64()*30, Y: 35 + rng.Float64()*30}
	ang := rng.Float64() * 2 * math.Pi
	curv := (rng.Float64() - 0.5) * 0.3

	numPts := 30 + rng.Intn(50)
	pts := make([]vec.Vec2, 0, numPts)
	pts = append(pts, pos)
	const step = 2.5
	for len(pts) < numPts {
		curv += (rng.Float64() - 0.5) * 0.15
		curv = max(-0.25, min(0.25, curv))
		if off := pos.Sub(centre); off.Length() > 15 {
			// steer towards the centre, keeping the turn gentle
			want := math.Atan2(-off.Y, -off.X)
			diff := math.Mod(want-ang+3*math.Pi, 2*math.Pi) - math.Pi
			curv = max(-0.25, min(0.25, diff))
		}
		ang += curv
		pos = pos.Add(vec.Vec2{X: step * math.Cos(ang), Y: step * math.Sin(ang)})
		pts = append(pts, pos)
	}
	return pts
}

// TestDotWindingUnderOverlap checks that a dot drawn with a round cap adds to
// the stroke where it overlaps other parts of the same stroke, instead of
// cancelling them.
//
// A dot is the one piece of a stroke outline built without a tangent
// direction, so its winding direction is chosen rather than inherited.  Wound
// against the rest, it would subtract itself from whatever it covers under the
// nonzero winding rule, leaving a hole.
func TestDotWindingUnderOverlap(t *testing.T) {
	const size = 40
	center := vec.Vec2{X: 20, Y: 20}

	tests := []struct {
		name string
		dash []float64
		path func() *path.Data
	}{
		{
			// a degenerate subpath sitting inside the stroke of a plain line
			name: "degenerate subpath",
			path: func() *path.Data {
				p := (&path.Data{}).MoveTo(vec.Vec2{X: 5, Y: 20})
				p.LineTo(vec.Vec2{X: 35, Y: 20})
				p.MoveTo(center)
				p.Close()
				return p
			},
		},
		{
			// a zero-length dash on one subpath, covered by a dash of another
			name: "zero-length dash",
			dash: []float64{0, 3, 8, 3},
			path: func() *path.Data {
				p := (&path.Data{}).MoveTo(center)
				p.LineTo(vec.Vec2{X: 38, Y: 20})
				p.MoveTo(vec.Vec2{X: 20, Y: 14})
				p.LineTo(vec.Vec2{X: 20, Y: 38})
				return p
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRasterizer(rect.Rect{URx: size, URy: size})
			r.Width = 10
			r.Cap = graphics.LineCapRound
			r.Join = graphics.LineJoinRound
			r.MiterLimit = 10
			r.Dash = tc.dash

			covered := make([]bool, size*size)
			r.Stroke(tc.path().Iter(), func(y, xMin int, coverage []float32) {
				for k, c := range coverage {
					covered[y*size+xMin+k] = c > 0.5
				}
			})

			// the dot alone covers this disc, whatever the rest of the path does
			d := r.Width/2 - 1
			for y := range size {
				for x := range size {
					c := vec.Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
					if c.Sub(center).Length() >= d {
						continue
					}
					if !covered[y*size+x] {
						t.Errorf("pixel (%d,%d) not covered", x, y)
					}
				}
			}
		})
	}
}
