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

// Package raster rasterises 2D vector paths to pixel coverage using the
// PDF/PostScript imaging model.
//
// The central type is a [Rasterizer], which holds the graphics parameters and
// provides three drawing methods:
//   - [Rasterizer.FillEvenOdd]: fills a path using the even-odd rule
//   - [Rasterizer.FillNonZero]: fills a path using the non-zero winding rule
//   - [Rasterizer.Stroke]: strokes a path with a given line style
//
// All drawing methods return pixel coverage via an emit callback function.
// The function is called once for each pixel row with non-zero coverage.
package raster

//go:generate go run ./testcases/genpdf
