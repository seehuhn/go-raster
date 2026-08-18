# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-raster is a 2D vector graphics rasterizer implementing the PDF/PostScript imaging model. It uses the signed-area coverage accumulation algorithm to convert vector paths to per-pixel coverage values.

## Build and Test Commands

```bash
go generate          # Generate PDFs and render reference images with Ghostscript
go test              # Run tests against reference images
go test -v           # Verbose output
go test -run TestAgainstReference  # Run the main comparison test
go test -run TestAgainstReference/fill_rect  # Run a single test case
```

The `go generate` pipeline requires Ghostscript (`gs`).

## Architecture

### Processing Pipelines

Fill pipeline:
```
Path (user space) → Flatten curves → Transform to device space → Rasterize → Coverage output
```

Stroke pipeline:
```
Path (user space) → Flatten curves → Expand to convex pieces (caps/joins/dashes) → Transform → Rasterize
```

### Core Algorithm

The rasterizer computes signed trapezoidal area contributions per pixel:
- Cover value: signed vertical extent of edge crossing pixel
- Area value: horizontal position of crossing within pixel
- Final coverage uses nonzero winding or even-odd fill rule

### Key Files

- `raster.go` — Rasterizer struct, curve flattening, edge accumulation, fill methods
- `stroke.go` — Stroke expansion: caps, joins, dashes, outline assembly
- `docs/specification.md` — Complete algorithm specification

### Public API

- `NewRasterizer(clip)` — Create rasterizer with clip bounds
- `FillNonZero(path, emit)` — Fill using nonzero winding rule
- `FillEvenOdd(path, emit)` — Fill using even-odd rule
- `Stroke(path, emit)` — Stroke using Width, Cap, Join, MiterLimit, Dash, DashPhase

### Dependencies

- `seehuhn.de/go/geom` — path.Data, vec.Vec2, matrix.Matrix, rect.Rect
- `seehuhn.de/go/pdf/graphics` — LineCapStyle, LineJoinStyle enums

## Test Infrastructure

Tests compare rendered output against Ghostscript-rendered reference images in
`testdata/reference/`. The pass/fail tolerance is defined by the percentile
thresholds in `compareImages` (raster_test.go).

On failure: a 3-panel image is written to `debug/`: actual output (left),
diff (middle), reference (right). In the diff panel green marks under-production
(expected > actual) and red marks over-production (expected < actual).

### Adding Test Cases

1. Add case to appropriate file in `testcases/` (fill.go, stroke.go, curve.go, dash.go, precision.go, complex.go, subpath.go, ctm.go, large.go)
2. Run `go generate` to regenerate references
3. Run `go test` to verify
