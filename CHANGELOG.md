# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
