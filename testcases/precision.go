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
	"seehuhn.de/go/geom/path"
)

var precisionCases = []TestCase{
	// Section 6.1: Subpixel Positioning
	{
		Name:   "subpixel_offset_00",
		Path:   offsetRectangle(20, 20, 24, 24, 0.0),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_offset_25",
		Path:   offsetRectangle(20, 20, 24, 24, 0.25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_offset_50",
		Path:   offsetRectangle(20, 20, 24, 24, 0.5),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_offset_75",
		Path:   offsetRectangle(20, 20, 24, 24, 0.75),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "thin_line_y_integer",
		Path:   horizontalLineAt(5, 10.0, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      1.0,
			Cap:        path.CapButt,
			Join:       path.JoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "thin_line_y_half",
		Path:   horizontalLineAt(5, 10.5, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      1.0,
			Cap:        path.CapButt,
			Join:       path.JoinMiter,
			MiterLimit: 10,
		},
	},

	// Section 6.2: Large Coordinates
	{
		Name:   "large_coord_centered",
		Path:   largeOffsetRectangle(1000, 1000, 20),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "small_shape_large_offset",
		Path:   smallShapeAtLargeOffset(10000, 10000, 2),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "float64_precision",
		Path:   float64PrecisionShape(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
}

// offsetRectangle builds a rectangular path with a subpixel offset applied to all coordinates.
func offsetRectangle(x1, y1, w, h, offset float64) *path.Data {
	ox1 := x1 + offset
	oy1 := y1 + offset
	ox2 := x1 + w + offset
	oy2 := y1 + h + offset

	return (&path.Data{}).
		MoveTo(pt(ox1, oy1)).
		LineTo(pt(ox2, oy1)).
		LineTo(pt(ox2, oy2)).
		LineTo(pt(ox1, oy2)).
		Close()
}

// horizontalLineAt builds a horizontal line segment at a specific y position.
func horizontalLineAt(x1, y, x2 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y)).
		LineTo(pt(x2, y))
}

// largeOffsetRectangle builds a rectangle centered at large coordinates,
// but translated back to fit the canvas. This tests precision at large offsets.
func largeOffsetRectangle(cx, cy, size float64) *path.Data {
	// Translate the large coordinates back to canvas center (32, 32)
	// The shape is computed at (cx, cy) then shifted to canvas
	translateX := 32 - cx
	translateY := 32 - cy

	x1 := cx - size/2 + translateX
	y1 := cy - size/2 + translateY
	x2 := cx + size/2 + translateX
	y2 := cy + size/2 + translateY

	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x1, y2)).
		Close()
}

// smallShapeAtLargeOffset builds a tiny shape at a large offset,
// translated back to the canvas for rendering.
func smallShapeAtLargeOffset(cx, cy, size float64) *path.Data {
	// The shape is logically at (cx, cy) but we translate it to canvas
	translateX := 32 - cx
	translateY := 32 - cy

	x1 := cx - size/2 + translateX
	y1 := cy - size/2 + translateY
	x2 := cx + size/2 + translateX
	y2 := cy + size/2 + translateY

	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x1, y2)).
		Close()
}

// float64PrecisionShape builds a shape using coordinates that require
// full float64 precision to represent accurately.
func float64PrecisionShape() *path.Data {
	// Use coordinates with many significant digits
	// These values differ only in the low bits of float64
	base := 32.0
	delta1 := 0.123456789012345
	delta2 := 0.123456789012346

	x1 := base - 10 + delta1
	y1 := base - 10 + delta1
	x2 := base + 10 + delta2
	y2 := base + 10 + delta2

	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x1, y2)).
		Close()
}
