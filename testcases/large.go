// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package testcases

import (
	"seehuhn.de/go/geom/path"
)

// largeCases contains test cases with bounding boxes > 65536 pixels
// to exercise Approach B (active edge list) in the rasterizer.
var largeCases = []TestCase{
	// Simple large rectangle - tests basic Approach B functionality
	{
		Name:   "large_rectangle",
		Path:   rectangle(50, 50, 462, 462),
		Width:  512,
		Height: 512,
		Op:     Fill{Rule: NonZero},
	},

	// Large concentric rectangles - tests winding rules with Approach B
	{
		Name:   "large_concentric_nonzero",
		Path:   concentricRectangles(256, 256, 200, 100),
		Width:  512,
		Height: 512,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "large_concentric_evenodd",
		Path:   concentricRectangles(256, 256, 200, 100),
		Width:  512,
		Height: 512,
		Op:     Fill{Rule: EvenOdd},
	},

	// Large diamond (diagonal edges) - tests sloped edges with Approach B
	{
		Name:   "large_diamond",
		Path:   diamond(256, 256, 180),
		Width:  512,
		Height: 512,
		Op:     Fill{Rule: NonZero},
	},

	// Grid of rectangles - tests many subpaths with Approach B
	{
		Name:   "large_grid",
		Path:   rectangleGrid(8, 8, 512, 512, 4),
		Width:  512,
		Height: 512,
		Op:     Fill{Rule: NonZero},
	},

	// Large shape that extends outside clip bounds - tests clipping with Approach B
	{
		Name:   "large_clipped",
		Path:   rectangle(-100, 100, 612, 400),
		Width:  512,
		Height: 512,
		Op:     Fill{Rule: NonZero},
	},
}

// rectangleGrid builds a grid of rectangles.
func rectangleGrid(rows, cols, width, height int, gap float64) *path.Data {
	cellW := float64(width) / float64(cols)
	cellH := float64(height) / float64(rows)

	p := &path.Data{}
	for row := range rows {
		for col := range cols {
			x1 := float64(col)*cellW + gap
			y1 := float64(row)*cellH + gap
			x2 := float64(col+1)*cellW - gap
			y2 := float64(row+1)*cellH - gap

			p = p.
				MoveTo(pt(x1, y1)).
				LineTo(pt(x2, y1)).
				LineTo(pt(x2, y2)).
				LineTo(pt(x1, y2)).
				Close()
		}
	}

	return p
}
