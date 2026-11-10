# go-raster

A 2D vector graphics rasterizer for Go, implementing the PDF/PostScript imaging model.

## Overview

go-raster converts vector paths to pixel coverage values—the fraction of each pixel covered by a filled or stroked path. It produces anti-aliased output without supersampling, suitable for rendering page graphics and font glyphs.

Features:

- Fill paths using nonzero winding or even-odd rules
- Stroke paths with configurable width, caps, joins, miter limit, and dash patterns
- Quadratic and cubic Bézier curve flattening with CTM-aware tolerance
- Zero allocations in steady state through buffer reuse

## Installation

```
go get seehuhn.de/go/raster
```

## Usage

```go
clip := image.Rect(0, 0, 100, 100)
r := raster.NewRasterizer(clip)

r.CTM = matrix.Scale(2, 2) // optional transform

var p path.Path // iterator over path segments
r.FillNonZero(p, func(y, xMin int, coverage []float32) {
    // coverage[i] is the coverage for pixel (xMin+i, y)
    // range: 0.0 (outside) to 1.0 (inside)
})
```

## Authors

Jochen Voss and Claude (Anthropic).

This code was nearly exclusively written by AI.

## Licence

This module is distributed under the BSD 3-Clause License; see the file LICENSE for the full text.
