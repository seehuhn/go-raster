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

package testcases

import (
	"math"

	"seehuhn.de/go/geom/path"
)

var complexCases = []TestCase{
	// Section 5.2: Mixed Operations
	{
		Name:   "mixed_lines_curves",
		Path:   mixedLinesCurves(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "stroked_mixed",
		Path:   mixedLinesCurves(),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      3,
			Cap:        path.CapRound,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "glyph_like",
		Path:   glyphLikeShape(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 5.3: Stroke Self-Intersection
	{
		Name:   "spiral_overlap",
		Path:   spiralPath(32, 32, 5, 25, 3),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        path.CapRound,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "figure_eight",
		Path:   figureEightStroke(32, 32, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        path.CapRound,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "thick_tight_curve",
		Path:   tightCurve(32, 32, 15),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      10,
			Cap:        path.CapRound,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "zigzag_thick",
		Path:   zigzagPath(10, 32, 54, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        path.CapRound,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
	},
}

// mixedLinesCurves builds a path combining line segments and Bezier curves.
func mixedLinesCurves() *path.Data {
	return (&path.Data{}).
		MoveTo(pt(10, 50)).
		LineTo(pt(20, 30)).                         // Line segment
		QuadTo(pt(32, 10), pt(44, 30)).             // Quadratic curve
		LineTo(pt(54, 50)).                         // Line segment
		CubeTo(pt(48, 60), pt(16, 60), pt(10, 50)). // Cubic curve back to start area
		Close()
}

// glyphLikeShape builds a complex shape similar to a typographic glyph.
// This resembles a simplified lowercase 'a' or 'e' character.
func glyphLikeShape() *path.Data {
	const kappa = 0.5522847498307936

	// Outer bowl (circular shape)
	cx, cy := 32.0, 38.0
	r := 18.0
	k := r * kappa

	// Start at right of bowl
	p := (&path.Data{}).
		MoveTo(pt(cx+r, cy)).
		CubeTo(pt(cx+r, cy-k), pt(cx+k, cy-r), pt(cx, cy-r)). // Top-right quadrant
		CubeTo(pt(cx-k, cy-r), pt(cx-r, cy-k), pt(cx-r, cy)). // Top-left quadrant
		CubeTo(pt(cx-r, cy+k), pt(cx-k, cy+r), pt(cx, cy+r)). // Bottom-left quadrant
		CubeTo(pt(cx+k, cy+r), pt(cx+r, cy+k), pt(cx+r, cy)). // Bottom-right quadrant
		LineTo(pt(cx+r, 10)).                                 // Stem going up
		LineTo(pt(cx+r-6, 10)).                               // Across top
		LineTo(pt(cx+r-6, cy))                                // Down to bowl

	// Inner counter (hole in the bowl)
	ir := 8.0
	ik := ir * kappa

	// Line to start of inner circle (moving inward)
	p = p.LineTo(pt(cx+ir, cy)).
		// Draw inner circle counter-clockwise (reverse winding for hole)
		CubeTo(pt(cx+ir, cy+ik), pt(cx+ik, cy+ir), pt(cx, cy+ir)).
		CubeTo(pt(cx-ik, cy+ir), pt(cx-ir, cy+ik), pt(cx-ir, cy)).
		CubeTo(pt(cx-ir, cy-ik), pt(cx-ik, cy-ir), pt(cx, cy-ir)).
		CubeTo(pt(cx+ik, cy-ir), pt(cx+ir, cy-ik), pt(cx+ir, cy)).
		Close()

	return p
}

// spiralPath builds an Archimedean spiral that overlaps itself.
func spiralPath(cx, cy, rMin, rMax float64, turns float64) *path.Data {
	// 32 segments per turn
	steps := max(int(turns*32), 8)

	totalAngle := turns * 2 * math.Pi
	rGrowth := (rMax - rMin) / totalAngle

	// Start point
	startX := cx + rMin
	startY := cy
	p := (&path.Data{}).MoveTo(pt(startX, startY))

	// Draw spiral with line segments
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		angle := t * totalAngle
		r := rMin + rGrowth*angle

		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)

		p = p.LineTo(pt(x, y))
	}

	return p
}

// figureEightStroke builds a figure-eight (lemniscate-like) path for stroke testing.
func figureEightStroke(cx, cy, size float64) *path.Data {
	// Using two circular arcs to form a figure-eight
	const kappa = 0.5522847498307936
	r := size / 2
	k := r * kappa

	// Top circle center
	topCy := cy - r/2

	// Start at center crossing point
	p := (&path.Data{}).
		MoveTo(pt(cx, cy)).
		// Upper loop (clockwise)
		CubeTo(pt(cx+k, cy-r/4), pt(cx+r, topCy-k/2), pt(cx+r, topCy)).
		CubeTo(pt(cx+r, topCy-k), pt(cx+k, topCy-r), pt(cx, topCy-r)).
		CubeTo(pt(cx-k, topCy-r), pt(cx-r, topCy-k), pt(cx-r, topCy)).
		CubeTo(pt(cx-r, topCy+k/2), pt(cx-k, cy-r/4), pt(cx, cy))

	// Bottom circle center
	botCy := cy + r/2

	// Lower loop (counter-clockwise to cross)
	p = p.
		CubeTo(pt(cx-k, cy+r/4), pt(cx-r, botCy-k/2), pt(cx-r, botCy)).
		CubeTo(pt(cx-r, botCy+k), pt(cx-k, botCy+r), pt(cx, botCy+r)).
		CubeTo(pt(cx+k, botCy+r), pt(cx+r, botCy+k), pt(cx+r, botCy)).
		CubeTo(pt(cx+r, botCy-k/2), pt(cx+k, cy+r/4), pt(cx, cy))

	return p
}

// tightCurve builds a U-shaped curve where the inner radius is small
// relative to stroke width, causing the inner edge to cross.
func tightCurve(cx, cy, size float64) *path.Data {
	const kappa = 0.5522847498307936
	r := size
	k := r * kappa

	return (&path.Data{}).
		MoveTo(pt(cx-r, cy-size)).
		LineTo(pt(cx-r, cy)).                                 // Down to curve start
		CubeTo(pt(cx-r, cy+k), pt(cx-k, cy+r), pt(cx, cy+r)). // Tight U-turn using cubic curves
		CubeTo(pt(cx+k, cy+r), pt(cx+r, cy+k), pt(cx+r, cy)).
		LineTo(pt(cx+r, cy-size)) // Up to end
}

// zigzagPath builds a zigzag pattern where adjacent thick strokes overlap.
func zigzagPath(x1, cy, x2, amplitude float64) *path.Data {
	segments := 5
	width := x2 - x1
	segWidth := width / float64(segments)

	p := (&path.Data{}).MoveTo(pt(x1, cy))

	for i := 1; i <= segments; i++ {
		x := x1 + float64(i)*segWidth
		var y float64
		if i%2 == 1 {
			y = cy - amplitude
		} else {
			y = cy + amplitude
		}
		p = p.LineTo(pt(x, y))
	}

	return p
}
