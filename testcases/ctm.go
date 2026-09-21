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

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
)

var ctmCases = []TestCase{
	// ========================================
	// Section 7.1: Uniform Scaling
	// ========================================
	{
		Name:   "scale_2x",
		Path:   rectangle(0, 0, 20, 20),
		Width:  128,
		Height: 128,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.Scale(2, 2).Translate(24, 24),
	},
	{
		Name:   "scale_half",
		Path:   rectangle(0, 0, 80, 80),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.Scale(0.5, 0.5).Translate(12, 12),
	},
	{
		Name:   "scale_10x",
		Path:   rectangle(0, 0, 4, 4),
		Width:  128,
		Height: 128,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.Scale(10, 10).Translate(44, 44),
	},

	// ========================================
	// Section 7.2: Rotation
	// ========================================
	{
		Name:   "rotate_45deg",
		Path:   rectangle(-10, -10, 10, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.RotateDeg(45).Translate(32, 32),
	},
	{
		Name:   "rotate_90deg",
		Path:   rectangle(-15, -10, 15, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.RotateDeg(90).Translate(32, 32),
	},
	{
		Name:   "rotate_5deg",
		Path:   rectangle(-20, -10, 20, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.RotateDeg(5).Translate(32, 32),
	},

	// ========================================
	// Section 7.3: Non-Uniform Scaling
	// ========================================
	{
		Name:   "scale_2x_1y",
		Path:   rectangle(-10, -10, 10, 10),
		Width:  128,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.Scale(2, 1).Translate(64, 32),
	},
	{
		Name:   "scale_1x_2y",
		Path:   rectangle(-10, -10, 10, 10),
		Width:  64,
		Height: 128,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.Scale(1, 2).Translate(32, 64),
	},
	{
		Name:   "circle_to_ellipse",
		Path:   circle(0, 0, 15),
		Width:  128,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		CTM:    matrix.Scale(2, 1).Translate(64, 32),
	},

	// ========================================
	// Section 7.4: Shear
	// ========================================
	{
		Name:   "shear_horizontal",
		Path:   rectangle(-15, -15, 15, 15),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		// Shear matrix: [1, 0, 0.5, 1, 0, 0] then translate
		CTM: matrix.Matrix{1, 0, 0.5, 1, 0, 0}.Translate(32, 32),
	},
	{
		Name:   "shear_vertical",
		Path:   rectangle(-15, -15, 15, 15),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		// Shear matrix: [1, 0.5, 0, 1, 0, 0] then translate
		CTM: matrix.Matrix{1, 0.5, 0, 1, 0, 0}.Translate(32, 32),
	},
	{
		Name:   "shear_and_rotate",
		Path:   rectangle(-12, -12, 12, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
		// Shear then rotate 30 degrees
		CTM: matrix.Matrix{1, 0, 0.3, 1, 0, 0}.RotateDeg(30).Translate(32, 32),
	},

	// ========================================
	// Section 7.5: Strokes Under Transform
	// ========================================
	{
		Name:   "round_cap_nonuniform",
		Path:   horizontalLineCentered(-20, 0, 20),
		Width:  128,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        path.CapRound,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
		// Non-uniform scale: round caps should become elliptical in device space
		CTM: matrix.Scale(2, 1).Translate(64, 32),
	},
	{
		Name:   "round_join_rotated",
		Path:   cornerCentered(0, 0, math.Pi/3),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        path.CapButt,
			Join:       path.JoinRound,
			MiterLimit: 10,
		},
		CTM: matrix.RotateDeg(30).Translate(32, 32),
	},
	{
		Name:   "dash_scaled",
		Path:   horizontalLineCentered(-25, 0, 25),
		Width:  128,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        path.CapButt,
			Join:       path.JoinMiter,
			MiterLimit: 10,
			Dash:       []float64{5, 3},
			DashPhase:  0,
		},
		// 2x scale: dash pattern should scale accordingly
		CTM: matrix.Scale(2, 1).Translate(64, 32),
	},
}

// horizontalLineCentered creates a horizontal line centered at origin.
func horizontalLineCentered(x1, y, x2 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y)).
		LineTo(pt(x2, y))
}

// cornerCentered creates a corner path centered at (cx, cy) with given angle.
func cornerCentered(cx, cy float64, angle float64) *path.Data {
	length := 20.0
	halfAngle := angle / 2
	// First arm extends up-left
	x1 := cx - length*math.Cos(halfAngle)
	y1 := cy - length*math.Sin(halfAngle)
	// Second arm extends up-right
	x2 := cx + length*math.Cos(halfAngle)
	y2 := cy - length*math.Sin(halfAngle)

	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(cx, cy)).
		LineTo(pt(x2, y2))
}
