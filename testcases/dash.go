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

var dashCases = []TestCase{
	// ==========================================================================
	// Section 3.1: Basic Patterns
	// ==========================================================================

	// Single-element pattern [10] (becomes [10, 10])
	{
		Name:   "dash_single_element",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10},
			DashPhase:  0,
		},
	},

	// Three-element pattern [5, 3, 8] (becomes [5, 3, 8, 5, 3, 8])
	{
		Name:   "dash_three_element",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{5, 3, 8},
			DashPhase:  0,
		},
	},

	// Long dash, short gap [20, 2]
	{
		Name:   "dash_long_short",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{20, 2},
			DashPhase:  0,
		},
	},

	// Short dash, long gap [2, 20]
	{
		Name:   "dash_short_long",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{2, 20},
			DashPhase:  0,
		},
	},

	// Equal dash and gap [10, 10]
	{
		Name:   "dash_equal",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 10},
			DashPhase:  0,
		},
	},

	// Many elements [2, 2, 6, 2, 2, 10]
	{
		Name:   "dash_many_elements",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{2, 2, 6, 2, 2, 10},
			DashPhase:  0,
		},
	},

	// ==========================================================================
	// Section 3.2: Dash Phase
	// ==========================================================================

	// Phase = 0 (default, starts with dash)
	{
		Name:   "dash_phase_zero",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  0,
		},
	},

	// Phase = half dash length (starts mid-dash)
	{
		Name:   "dash_phase_half",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  5, // half of dash length (10/2)
		},
	},

	// Phase = dash length (starts with gap)
	{
		Name:   "dash_phase_dash_len",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  10, // full dash length, starts at gap
		},
	},

	// Phase = pattern length (same as phase 0)
	{
		Name:   "dash_phase_pattern_len",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  15, // 10 + 5 = full pattern length
		},
	},

	// Negative phase (wraps around)
	{
		Name:   "dash_phase_negative",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  -5, // negative wraps around
		},
	},

	// Large negative phase
	{
		Name:   "dash_phase_large_neg",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  -30, // large negative (wraps multiple times)
		},
	},

	// ==========================================================================
	// Section 3.3: Zero-Length Dashes
	// ==========================================================================

	// Pattern [0, 5] with round cap (dots)
	{
		Name:   "dash_zero_round",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{0, 5},
			DashPhase:  0,
		},
	},

	// Pattern [0, 5] with butt cap (nothing visible from zero-length)
	{
		Name:   "dash_zero_butt",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{0, 5},
			DashPhase:  0,
		},
	},

	// Pattern [0, 5] with square cap (nothing visible)
	{
		Name:   "dash_zero_square",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{0, 5},
			DashPhase:  0,
		},
	},

	// Pattern [0, 5, 10, 5] (alternating dots and dashes)
	{
		Name:   "dash_zero_mixed",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{0, 5, 10, 5},
			DashPhase:  0,
		},
	},

	// Pattern [0, 5] with non-zero phase (phase must advance past zero-length element)
	{
		Name:   "dash_zero_phase_round",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{0, 5},
			DashPhase:  3,
		},
	},

	// Pattern [0, 5, 10, 5] with non-zero phase (phase advances past zero-length element)
	{
		Name:   "dash_zero_mixed_phase",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{0, 5, 10, 5},
			DashPhase:  7,
		},
	},

	// ==========================================================================
	// Section 3.4: Dashes at Corners
	// ==========================================================================

	// Corner falls within dash - join should appear
	{
		Name:   "dash_corner_in_dash",
		Path:   corner(10, 50, 32, 20, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{40, 5},
			DashPhase:  0,
		},
	},

	// Corner falls within gap - no join visible
	{
		Name:   "dash_corner_in_gap",
		Path:   corner(10, 50, 32, 20, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{5, 40},
			DashPhase:  20, // phase puts corner in gap
		},
	},

	// Dash ends exactly at corner
	{
		Name:   "dash_end_at_corner",
		Path:   corner(10, 50, 32, 20, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{33, 10}, // first segment length ~33
			DashPhase:  0,
		},
	},

	// Dash starts exactly at corner
	{
		Name:   "dash_start_at_corner",
		Path:   corner(10, 50, 32, 20, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 23}, // gap ends at corner (~33)
			DashPhase:  0,
		},
	},

	// Very short dash at corner (dash < corner angle span)
	{
		Name:   "dash_short_at_corner",
		Path:   corner(10, 50, 32, 20, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{2, 8},
			DashPhase:  0,
		},
	},

	// ==========================================================================
	// Section 3.5: Overlapping Dash Segments (Boundary Case)
	// ==========================================================================

	// Thick dashed line with tight corner - width 10, dash [15, 5], 60deg corner
	{
		Name:   "dash_overlap_tight",
		Path:   cornerAngle(32, 50, 32, 32, 60),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      10,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{15, 5},
			DashPhase:  0,
		},
	},

	// Overlapping dash caps at corner - verify no visual artifacts
	{
		Name:   "dash_overlap_caps",
		Path:   cornerAngle(32, 54, 32, 32, 45),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  0,
		},
	},

	// Multiple corners within one dash pattern cycle
	{
		Name:   "dash_multi_corner",
		Path:   zigzag(10, 50, 22, 14, 32, 50, 42, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{50, 10},
			DashPhase:  0,
		},
	},

	// ==========================================================================
	// Section 3.6: Closed Path Dashing
	// ==========================================================================

	// Closed square, dashed - pattern restarts or wraps
	{
		Name:   "dash_closed_square",
		Path:   closedSquare(16, 16, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{10, 5},
			DashPhase:  0,
		},
	},

	// First and last dash connect - when both "on", should join not cap
	{
		Name:   "dash_closed_join",
		Path:   closedSquare(16, 16, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{32, 5}, // 32*4=128 perimeter, dash shows at start/end
			DashPhase:  0,
		},
	},

	// First dash "on", last dash "off" - cap at start, gap at end
	{
		Name:   "dash_closed_cap_gap",
		Path:   closedSquare(16, 16, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{20, 20},
			DashPhase:  10, // offset so end is in gap
		},
	},

	// Phase chosen so path starts and ends in same dash - proper join
	{
		Name:   "dash_closed_same_dash",
		Path:   closedSquare(16, 16, 32),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{64, 10}, // long dash to span multiple sides
			DashPhase:  32,                // phase puts us in same dash at start/end
		},
	},
}

// =============================================================================
// Helper functions for dash test cases
// =============================================================================

// cornerAngle builds a corner path with a specific angle.
// The first segment goes from (x1, y1) to (cx, cy), the second segment
// extends from (cx, cy) at the given angle (in degrees) from horizontal.
func cornerAngle(x1, y1, cx, cy float64, angleDeg float64) *path.Data {
	// Calculate second endpoint based on angle
	angleRad := angleDeg * math.Pi / 180
	length := 30.0 // segment length
	x2 := cx + length*math.Cos(angleRad)
	y2 := cy - length*math.Sin(angleRad) // y is inverted in screen coords

	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(cx, cy)).
		LineTo(pt(x2, y2))
}

// zigzag builds a zigzag path with multiple corners.
func zigzag(x1, y1, x2, y2, x3, y3, x4, y4, x5, y5 float64) *path.Data {
	return (&path.Data{}).
		MoveTo(pt(x1, y1)).
		LineTo(pt(x2, y2)).
		LineTo(pt(x3, y3)).
		LineTo(pt(x4, y4)).
		LineTo(pt(x5, y5))
}
