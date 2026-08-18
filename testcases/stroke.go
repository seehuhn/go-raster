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
	"seehuhn.de/go/pdf/graphics"
)

var strokeCases = []TestCase{
	{
		Name:   "line_butt",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "line_round",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "line_square",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "corner_miter",
		Path:   corner(10, 50, 32, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "corner_round",
		Path:   corner(10, 50, 32, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "corner_bevel",
		Path:   corner(10, 50, 32, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},
	{
		Name:   "dashed",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{8, 4},
			DashPhase:  0,
		},
	},

	// ========================================
	// Section 2.1: Line Width
	// ========================================

	{
		Name:   "width_thin_0_5",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      0.5,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "width_thin_1_0",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      1.0,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "width_thick_20",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      20,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "width_equals_dash",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{8, 4},
			DashPhase:  0,
		},
	},

	// ========================================
	// Section 2.2: Line Caps
	// ========================================

	// Caps on vertical line
	{
		Name:   "cap_vertical_butt",
		Path:   verticalLine(32, 10, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_vertical_round",
		Path:   verticalLine(32, 10, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_vertical_square",
		Path:   verticalLine(32, 10, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Caps on 45° diagonal line
	{
		Name:   "cap_diagonal_45_butt",
		Path:   diagonalLine(32, 32, 45, 30),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_diagonal_45_round",
		Path:   diagonalLine(32, 32, 45, 30),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_diagonal_45_square",
		Path:   diagonalLine(32, 32, 45, 30),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Caps on 30° diagonal line (asymmetric)
	{
		Name:   "cap_diagonal_30_butt",
		Path:   diagonalLine(32, 32, 30, 30),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_diagonal_30_round",
		Path:   diagonalLine(32, 32, 30, 30),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_diagonal_30_square",
		Path:   diagonalLine(32, 32, 30, 30),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Caps on very short segment (length < width)
	{
		Name:   "cap_short_butt",
		Path:   horizontalLine(30, 32, 34),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_short_round",
		Path:   horizontalLine(30, 32, 34),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_short_square",
		Path:   horizontalLine(30, 32, 34),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Caps where segment length equals width
	{
		Name:   "cap_length_equals_width_butt",
		Path:   horizontalLine(28, 32, 36),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_length_equals_width_round",
		Path:   horizontalLine(28, 32, 36),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_length_equals_width_square",
		Path:   horizontalLine(28, 32, 36),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Zero-length subpath: single-point closed path (MoveTo + Close)
	// Per PDF spec: round caps produce filled circle, others produce nothing
	{
		Name:   "cap_zero_closed_round",
		Path:   zeroLengthClosedPath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_zero_closed_butt",
		Path:   zeroLengthClosedPath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_zero_closed_square",
		Path:   zeroLengthClosedPath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Zero-length subpath: two points at same coordinates (MoveTo + LineTo same point)
	// Per PDF spec: round caps produce filled circle, others produce nothing
	{
		Name:   "cap_zero_line_round",
		Path:   zeroLengthLinePath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_zero_line_butt",
		Path:   zeroLengthLinePath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_zero_line_square",
		Path:   zeroLengthLinePath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Single-point open subpath (trailing MoveTo, no Close)
	// Per PDF spec: "A single-point open subpath (specified by a trailing m operator)
	// shall produce no output." - regardless of cap style
	{
		Name:   "cap_trailing_moveto_round",
		Path:   trailingMoveToPath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_trailing_moveto_butt",
		Path:   trailingMoveToPath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cap_trailing_moveto_square",
		Path:   trailingMoveToPath(32, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// ========================================
	// Section 2.3: Line Joins
	// ========================================

	// Join at 90° (right angle)
	{
		Name:   "join_90deg_miter",
		Path:   cornerAtAngle(32, 32, 90, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_90deg_round",
		Path:   cornerAtAngle(32, 32, 90, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_90deg_bevel",
		Path:   cornerAtAngle(32, 32, 90, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Join at 120° (obtuse)
	{
		Name:   "join_120deg_miter",
		Path:   cornerAtAngle(32, 32, 120, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_120deg_round",
		Path:   cornerAtAngle(32, 32, 120, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_120deg_bevel",
		Path:   cornerAtAngle(32, 32, 120, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Join at 45° (acute)
	{
		Name:   "join_45deg_miter",
		Path:   cornerAtAngle(32, 32, 45, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_45deg_round",
		Path:   cornerAtAngle(32, 32, 45, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_45deg_bevel",
		Path:   cornerAtAngle(32, 32, 45, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Join at 30° (sharp acute)
	{
		Name:   "join_30deg_miter",
		Path:   cornerAtAngle(32, 32, 30, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_30deg_round",
		Path:   cornerAtAngle(32, 32, 30, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_30deg_bevel",
		Path:   cornerAtAngle(32, 32, 30, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Join at 15° (very sharp, miter limit test)
	{
		Name:   "join_15deg_miter",
		Path:   cornerAtAngle(32, 32, 15, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_15deg_round",
		Path:   cornerAtAngle(32, 32, 15, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_15deg_bevel",
		Path:   cornerAtAngle(32, 32, 15, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Join at 170° (near-straight, almost no join visible)
	{
		Name:   "join_170deg_miter",
		Path:   cornerAtAngle(32, 32, 170, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_170deg_round",
		Path:   cornerAtAngle(32, 32, 170, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_170deg_bevel",
		Path:   cornerAtAngle(32, 32, 170, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Join at 179° (near-cusp)
	{
		Name:   "join_179deg_miter",
		Path:   cornerAtAngle(32, 32, 179, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_179deg_round",
		Path:   cornerAtAngle(32, 32, 179, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_179deg_bevel",
		Path:   cornerAtAngle(32, 32, 179, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Left turn vs right turn (both directions)
	{
		Name:   "join_left_turn_miter",
		Path:   leftTurnCorner(32, 32, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_right_turn_miter",
		Path:   rightTurnCorner(32, 32, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Three-segment path (two consecutive joins)
	{
		Name:   "join_three_segment_miter",
		Path:   threeSegmentPath(8, 48, 28, 16, 48, 48, 56, 16),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_three_segment_round",
		Path:   threeSegmentPath(8, 48, 28, 16, 48, 48, 56, 16),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_three_segment_bevel",
		Path:   threeSegmentPath(8, 48, 28, 16, 48, 48, 56, 16),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Closed triangle (three joins, no caps)
	{
		Name:   "join_closed_triangle_miter",
		Path:   closedTriangle(32, 12, 12, 52, 52, 52),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_closed_triangle_round",
		Path:   closedTriangle(32, 12, 12, 52, 52, 52),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_closed_triangle_bevel",
		Path:   closedTriangle(32, 12, 12, 52, 52, 52),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// Closed square (four 90° joins)
	{
		Name:   "join_closed_square_miter",
		Path:   closedSquare(12, 12, 40),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_closed_square_round",
		Path:   closedSquare(12, 12, 40),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "join_closed_square_bevel",
		Path:   closedSquare(12, 12, 40),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},

	// ========================================
	// Section 2.4: Miter Limit
	// ========================================

	// Miter limit 1.0 (always bevel)
	{
		Name:   "miter_limit_1_0",
		Path:   cornerAtAngle(32, 32, 60, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 1.0,
		},
	},

	// Miter limit 1.414 (bevel below 90°)
	{
		Name:   "miter_limit_1_414",
		Path:   cornerAtAngle(32, 32, 60, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 1.414,
		},
	},

	// Miter limit 2.0 (bevel below 60°)
	{
		Name:   "miter_limit_2_0",
		Path:   cornerAtAngle(32, 32, 60, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 2.0,
		},
	},

	// Angle just below miter cutoff (shows miter) - 30° angle, limit ~3.86
	{
		Name:   "miter_just_below_cutoff",
		Path:   cornerAtAngle(32, 32, 30, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 4.0,
		},
	},

	// Angle just above miter cutoff (shows bevel) - 30° angle, limit ~3.86
	{
		Name:   "miter_just_above_cutoff",
		Path:   cornerAtAngle(32, 32, 30, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 3.5,
		},
	},

	// Same angle with different miter limits (comparison) - 45° angle
	{
		Name:   "miter_45deg_limit_1_5",
		Path:   cornerAtAngle(32, 32, 45, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 1.5,
		},
	},
	{
		Name:   "miter_45deg_limit_3_0",
		Path:   cornerAtAngle(32, 32, 45, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 3.0,
		},
	},

	// ========================================
	// Section 2.5: Cusp Handling
	// ========================================

	// 180° turn (exact reversal) — should show two caps
	{
		Name:   "cusp_180deg_round",
		Path:   cornerAtAngle(32, 32, 180, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cusp_180deg_butt",
		Path:   cornerAtAngle(32, 32, 180, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},

	// 179.5° turn — should show two caps (cusp threshold)
	{
		Name:   "cusp_179_5deg_round",
		Path:   cornerAtAngle(32, 32, 179.5, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cusp_179_5deg_butt",
		Path:   cornerAtAngle(32, 32, 179.5, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},

	// 178° turn — should show normal join
	{
		Name:   "cusp_178deg_round",
		Path:   cornerAtAngle(32, 32, 178, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cusp_178deg_miter",
		Path:   cornerAtAngle(32, 32, 178, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Narrow folds where the inner offset lines meet beyond the ends of the
	// adjacent segments.  The stroke must stay within half a line width of
	// the path.
	{
		Name:   "cusp_fold_left_round",
		Path:   corner(20, 12, 20, 52, 23, 16),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cusp_fold_right_round",
		Path:   corner(44, 12, 44, 52, 41, 16),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cusp_fold_closed_round",
		Path:   closedTriangle(32, 50, 34, 14, 36, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
}

// horizontalLine builds a horizontal line segment.
func horizontalLine(x1, y, x2 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y)).
		LineTo(pt(x2, y))
}

// corner builds a path with two line segments meeting at a corner.
func corner(x1, y1, x2, y2, x3, y3 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x3, y3))
}

// verticalLine builds a vertical line segment.
func verticalLine(x, y1, y2 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x, y1)).
		LineTo(pt(x, y2))
}

// diagonalLine builds a diagonal line at a given angle (in degrees) from center.
// The line extends length/2 in each direction from the center point.
func diagonalLine(cx, cy, angleDeg, length float64) *path.Data {
	angleRad := angleDeg * math.Pi / 180
	dx := math.Cos(angleRad) * length / 2
	dy := math.Sin(angleRad) * length / 2
	return (&path.Data{}).
		MoveTo(pt(cx-dx, cy-dy)).
		LineTo(pt(cx+dx, cy+dy))
}

// zeroLengthClosedPath builds a single-point closed path (MoveTo + Close).
// Per PDF spec, this is a degenerate subpath that should produce a filled
// circle with round caps.
func zeroLengthClosedPath(x, y float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x, y)).
		Close()
}

// zeroLengthLinePath builds a path with two points at the same coordinates
// (MoveTo + LineTo to same point). Per PDF spec, this is a degenerate subpath
// that should produce a filled circle with round caps.
func zeroLengthLinePath(x, y float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x, y)).
		LineTo(pt(x, y))
}

// trailingMoveToPath builds a single-point open subpath (just a MoveTo).
// Per PDF spec: "A single-point open subpath (specified by a trailing m operator)
// shall produce no output."
func trailingMoveToPath(x, y float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x, y))
}

// cornerAtAngle builds a corner at a specified angle (in degrees).
// The corner is centered at (cx, cy) with arms of the given length.
// The angle is the interior angle between the two line segments.
func cornerAtAngle(cx, cy, angleDeg, armLength float64) *path.Data {
	// First segment comes from the left horizontally
	// Second segment goes at the specified angle
	x1 := cx - armLength
	y1 := cy

	// End point (at the specified angle from horizontal)
	x3 := cx + armLength*math.Cos(math.Pi-angleDeg*math.Pi/180)
	y3 := cy + armLength*math.Sin(math.Pi-angleDeg*math.Pi/180)

	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(cx, cy)).
		LineTo(pt(x3, y3))
}

// leftTurnCorner builds a corner that turns left (counter-clockwise).
func leftTurnCorner(cx, cy, armLength float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(cx-armLength, cy)).
		LineTo(pt(cx, cy)).
		LineTo(pt(cx, cy-armLength))
}

// rightTurnCorner builds a corner that turns right (clockwise).
func rightTurnCorner(cx, cy, armLength float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(cx-armLength, cy)).
		LineTo(pt(cx, cy)).
		LineTo(pt(cx, cy+armLength))
}

// threeSegmentPath builds a path with three line segments (two corners).
func threeSegmentPath(x1, y1, x2, y2, x3, y3, x4, y4 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x3, y3)).
		LineTo(pt(x4, y4))
}

// closedTriangle builds a closed triangular path.
func closedTriangle(x1, y1, x2, y2, x3, y3 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x3, y3)).
		Close()
}

// closedSquare builds a closed square path starting at (x, y) with given side length.
func closedSquare(x, y, side float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x, y)).
		LineTo(pt(x+side, y)).
		LineTo(pt(x+side, y+side)).
		LineTo(pt(x, y+side)).
		Close()
}
