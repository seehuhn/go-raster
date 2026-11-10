// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package testcases

import (
	"math"

	"seehuhn.de/go/geom/path"
)

var fillCases = []TestCase{
	{
		Name:   "triangle_nonzero",
		Path:   triangle(10, 50, 32, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "triangle_evenodd",
		Path:   triangle(10, 50, 32, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "star_nonzero",
		Path:   fivePointStar(32, 32, 25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "star_evenodd",
		Path:   fivePointStar(32, 32, 25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "rectangle",
		Path:   rectangle(10, 10, 44, 44),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 1.1 Fill Rules
	{
		Name:   "concentric_rect_nonzero",
		Path:   concentricRectangles(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "concentric_rect_evenodd",
		Path:   concentricRectangles(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "overlapping_circles_nonzero",
		Path:   overlappingCircles(24, 32, 44, 32, 16),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "overlapping_circles_evenodd",
		Path:   overlappingCircles(24, 32, 44, 32, 16),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "figure_eight_nonzero",
		Path:   figureEight(32, 32, 20, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "figure_eight_evenodd",
		Path:   figureEight(32, 32, 20, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "high_winding_nonzero",
		Path:   highWindingRect(32, 32, 20, 3),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "high_winding_evenodd",
		Path:   highWindingRect(32, 32, 20, 3),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "alternating_winding_nonzero",
		Path:   alternatingWinding(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "alternating_winding_evenodd",
		Path:   alternatingWinding(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},

	// Section 1.2 Edge Cases
	{
		Name:   "horizontal_edges",
		Path:   rectangle(10, 20, 54, 44),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "vertical_edges",
		Path:   rectangle(28, 5, 36, 59),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "diagonal_45deg",
		Path:   diamond(32, 32, 20),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "near_horizontal",
		Path:   nearHorizontalQuad(10, 30, 54, 30.4),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "near_vertical",
		Path:   nearVerticalQuad(30, 10, 30.4, 54),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "single_pixel",
		Path:   triangle(30, 32, 32, 30, 34, 32),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_shape",
		Path:   triangle(31.2, 31.8, 31.5, 31.2, 31.8, 31.8),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 1.3 Boundary Conditions
	{
		Name:   "touching_edge",
		Path:   rectangle(0, 10, 54, 54),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "partially_clipped",
		Path:   rectangle(-10, 20, 40, 74),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "fully_outside",
		Path:   rectangle(70, 70, 100, 100),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "pixel_aligned",
		Path:   rectangle(10, 10, 50, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "half_pixel_offset",
		Path:   rectangle(10.5, 10.5, 50.5, 50.5),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "clipped_nested_rects",
		Path:   clippedNestedRects(),
		Width:  32,
		Height: 32,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "mixed_close",
		Path:   mixedClose(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
}

// triangle builds a triangular path.
func triangle(x1, y1, x2, y2, x3, y3 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x3, y3)).
		Close()
}

// fivePointStar builds a five-pointed star (self-intersecting).
func fivePointStar(cx, cy, r float64) *path.Data {
	// five points, connecting every second point
	var pts [5]struct{ x, y float64 }
	for i := range 5 {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		pts[i].x = cx + r*math.Cos(angle)
		pts[i].y = cy + r*math.Sin(angle)
	}

	// draw star: 0 -> 2 -> 4 -> 1 -> 3 -> 0
	order := [5]int{0, 2, 4, 1, 3}
	p := (&path.Data{}).MoveTo(pt(pts[order[0]].x, pts[order[0]].y))
	for _, i := range order[1:] {
		p = p.LineTo(pt(pts[i].x, pts[i].y))
	}
	return p.Close()
}

// rectangle builds a rectangular path.
func rectangle(x1, y1, x2, y2 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x1, y2)).
		Close()
}

// concentricRectangles builds two nested rectangles (outer clockwise, inner counter-clockwise).
func concentricRectangles(cx, cy, outerSize, innerSize float64) *path.Data {
	// Outer rectangle (clockwise)
	p := (&path.Data{}).
		MoveTo(pt(cx-outerSize, cy-outerSize)).
		LineTo(pt(cx+outerSize, cy-outerSize)).
		LineTo(pt(cx+outerSize, cy+outerSize)).
		LineTo(pt(cx-outerSize, cy+outerSize)).
		Close()

	// Inner rectangle (counter-clockwise for hole)
	return p.
		MoveTo(pt(cx-innerSize, cy-innerSize)).
		LineTo(pt(cx-innerSize, cy+innerSize)).
		LineTo(pt(cx+innerSize, cy+innerSize)).
		LineTo(pt(cx+innerSize, cy-innerSize)).
		Close()
}

// overlappingCircles builds two overlapping circles using cubic Bezier approximation.
func overlappingCircles(cx1, cy1 float64, cx2, cy2 float64, r float64) *path.Data {
	const kappa = 0.5522847498307936

	// First circle
	k := r * kappa
	p := (&path.Data{}).
		MoveTo(pt(cx1+r, cy1)).
		CubeTo(pt(cx1+r, cy1-k), pt(cx1+k, cy1-r), pt(cx1, cy1-r)).
		CubeTo(pt(cx1-k, cy1-r), pt(cx1-r, cy1-k), pt(cx1-r, cy1)).
		CubeTo(pt(cx1-r, cy1+k), pt(cx1-k, cy1+r), pt(cx1, cy1+r)).
		CubeTo(pt(cx1+k, cy1+r), pt(cx1+r, cy1+k), pt(cx1+r, cy1)).
		Close()

	// Second circle
	return p.
		MoveTo(pt(cx2+r, cy2)).
		CubeTo(pt(cx2+r, cy2-k), pt(cx2+k, cy2-r), pt(cx2, cy2-r)).
		CubeTo(pt(cx2-k, cy2-r), pt(cx2-r, cy2-k), pt(cx2-r, cy2)).
		CubeTo(pt(cx2-r, cy2+k), pt(cx2-k, cy2+r), pt(cx2, cy2+r)).
		CubeTo(pt(cx2+k, cy2+r), pt(cx2+r, cy2+k), pt(cx2+r, cy2)).
		Close()
}

// figureEight builds a self-crossing figure-eight shape.
func figureEight(cx, cy, width, height float64) *path.Data {
	// Start at top-left, cross through center
	return (&path.Data{}).
		MoveTo(pt(cx-width, cy-height)).
		LineTo(pt(cx+width, cy+height)). // Go to bottom-right
		LineTo(pt(cx+width, cy-height)). // Go to top-right
		LineTo(pt(cx-width, cy+height)). // Go to bottom-left (crossing the first line)
		Close()
}

// highWindingRect builds a rectangle wound multiple times in the same direction.
func highWindingRect(cx, cy, size float64, windings int) *path.Data {
	p := &path.Data{}
	for range windings {
		p = p.
			MoveTo(pt(cx-size, cy-size)).
			LineTo(pt(cx+size, cy-size)).
			LineTo(pt(cx+size, cy+size)).
			LineTo(pt(cx-size, cy+size)).
			Close()
	}
	return p
}

// alternatingWinding builds a clockwise outer rectangle with a clockwise inner rectangle.
func alternatingWinding(cx, cy, outerSize, innerSize float64) *path.Data {
	// Outer rectangle (counter-clockwise)
	p := (&path.Data{}).
		MoveTo(pt(cx-outerSize, cy-outerSize)).
		LineTo(pt(cx-outerSize, cy+outerSize)).
		LineTo(pt(cx+outerSize, cy+outerSize)).
		LineTo(pt(cx+outerSize, cy-outerSize)).
		Close()

	// Inner rectangle (clockwise - same direction conceptually)
	return p.
		MoveTo(pt(cx-innerSize, cy-innerSize)).
		LineTo(pt(cx+innerSize, cy-innerSize)).
		LineTo(pt(cx+innerSize, cy+innerSize)).
		LineTo(pt(cx-innerSize, cy+innerSize)).
		Close()
}

// diamond builds a diamond shape (45 degree rotated square).
func diamond(cx, cy, size float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(cx, cy-size)).
		LineTo(pt(cx+size, cy)).
		LineTo(pt(cx, cy+size)).
		LineTo(pt(cx-size, cy)).
		Close()
}

// nearHorizontalQuad builds a quadrilateral with near-horizontal top/bottom edges.
func nearHorizontalQuad(x1, y1, x2, y2 float64) *path.Data {
	height := 10.0
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x2, y2+height)).
		LineTo(pt(x1, y1+height)).
		Close()
}

// nearVerticalQuad builds a quadrilateral with near-vertical left/right edges.
func nearVerticalQuad(x1, y1, x2, y2 float64) *path.Data {
	width := 10.0
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x1+width, y1)).
		LineTo(pt(x2+width, y2)).
		LineTo(pt(x2, y2)).
		Close()
}

// clippedNestedRects builds two nested rectangles that extend outside the clip region.
// Outer: (-4,-8)--(36,-8)--(36,28)--(-4,28)--close (clockwise)
// Inner: (4,-4)--(4,20)--(28,20)--(28,-4)--close (counter-clockwise for hole)
func clippedNestedRects() *path.Data {
	// outer rectangle (clockwise)
	p := (&path.Data{}).
		MoveTo(pt(-4, -8)).
		LineTo(pt(36, -8)).
		LineTo(pt(36, 28)).
		LineTo(pt(-4, 28)).
		Close()

	// inner rectangle (counter-clockwise for hole)
	return p.
		MoveTo(pt(4, -4)).
		LineTo(pt(4, 20)).
		LineTo(pt(28, 20)).
		LineTo(pt(28, -4)).
		Close()
}

// mixedClose builds two rectangles: first without explicit close, second with close.
func mixedClose() *path.Data {
	// first rectangle (no close)
	p := (&path.Data{}).
		MoveTo(pt(2, 2)).
		LineTo(pt(30, 2)).
		LineTo(pt(30, 30)).
		LineTo(pt(2, 30))

	// second rectangle (with close)
	return p.
		MoveTo(pt(34, 34)).
		LineTo(pt(62, 34)).
		LineTo(pt(62, 62)).
		LineTo(pt(34, 62)).
		Close()
}
