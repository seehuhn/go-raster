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
	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// TestCase defines a single rendering test.
type TestCase struct {
	Name   string        // lowercase a-z and _ only
	Path   *path.Data    // the geometry to render
	Width  int           // canvas width in pixels
	Height int           // canvas height in pixels
	Op     Operation     // fill or stroke
	CTM    matrix.Matrix // transformation matrix (zero-value means no transform)
}

// Operation is the rendering operation to apply to the path.
type Operation interface {
	isOperation()
}

// FillRule specifies the rule for determining interior points.
type FillRule int

const (
	NonZero FillRule = iota
	EvenOdd
)

// Fill specifies a fill operation.
type Fill struct {
	Rule FillRule
}

func (Fill) isOperation() {}

// Stroke specifies a stroke operation.
type Stroke struct {
	Width      float64        // line width (>0)
	Cap        path.CapStyle  // CapButt, CapRound, CapSquare
	Join       path.JoinStyle // JoinMiter, JoinRound, JoinBevel
	MiterLimit float64        // miter limit
	Dash       []float64      // dash pattern (nil for solid)
	DashPhase  float64        // dash phase offset
}

func (Stroke) isOperation() {}

// pt is a helper to create a vec.Vec2 from x, y coordinates.
func pt(x, y float64) vec.Vec2 {
	return vec.Vec2{X: x, Y: y}
}
