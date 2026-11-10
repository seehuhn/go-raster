// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

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
