# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.7.5] (2026-10-10)

### Added
- `DefaultFlatness` exports the curve-flattening tolerance a new
  `Rasterizer` starts with.

### Changed
- `Rasterizer.Clip` is now an `image.Rectangle` instead of a
  `rect.Rect`, matching the integer-aligned, half-open pixel rectangle
  it has always required.  This is a breaking API change.
- `Rasterizer.Cap` and `Join` now take `path.CapStyle` and
  `path.JoinStyle` from the geom module instead of
  `graphics.LineCapStyle`/`LineJoinStyle` from go-pdf, with
  `path.DefaultMiterLimit` as the default miter limit.  The package no
  longer imports go-pdf.  This is a breaking API change.
- Faster filling of paths with many edges or a wide clip rectangle:
  edges are grouped by scanline instead of sorted, and each row is
  integrated only over the pixels its active edges touch.

### Fixed
- Strokes no longer leave a hole where they cover the same area twice
  (a tight curve, a self-crossing path, a dash landing on itself), and
  no longer produce an unbounded spike at a sharp corner: the outline
  is now built as a union of convex pieces instead of a single
  contour.  A round cap on a zero-length subpath or dash was
  wound the wrong way and could cancel an overlapping part of the same
  stroke; it now winds the same way as the rest of the outline.
- An edge far outside the clip rectangle, from an extreme CTM, could
  make the rasterizer walk its entire span and effectively hang.
  Edges are now clipped to the clip rectangle as they are collected,
  and an edge whose horizontal span overflows relative to its vertical
  extent is treated as horizontal instead of producing `NaN`.
- A dashed subpath entirely covered by one dash stayed open and was
  capped at both ends instead of closed with a join.
- A dash boundary landing near a path vertex could fall on either side
  of it depending on rounding, pointing a cap along the wrong segment
  or leaving a sliver of the next segment at the start of a dash.
  Boundaries within a small tolerance of a vertex now snap onto it.

## [v0.7.4] (2026-06-25)

### Changed
- `Flatness` and `Width` are validated; invalid values are rejected.

### Fixed
- Curve, arc, and dash subdivision is bounded against malformed input.

## [0.7.3] (2026-05-19)

### Changed
- Dependency versions bumped to match the v0.7.3 Quire release; no
  source changes.

## [0.7.2] (2026-05-11)

### Changed
- Minimum Go version raised to 1.25.

## [0.7.1] (2026-03-31)

### Changed
- Accept `path.Path` iterator instead of `*path.Data` for stroke and fill operations.
- Scanline integration optimised with bounds-check elimination and even-odd fast path.

### Fixed
- Dash phase advancement past zero-length dash elements.

## [0.7.0] (2025-01-25)

Initial tagged release.
