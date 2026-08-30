# CLAUDE.md

go-raster is a 2D vector graphics rasterizer implementing the PDF/PostScript
imaging model.  The algorithm is specified in `docs/specification.md`; the
public API is documented in the Go sources (`go doc seehuhn.de/go/raster`).

## Build and Test Commands

```bash
go generate                              # regenerate PDFs and Ghostscript reference images
go test                                  # run tests against reference images
go test -run TestAgainstReference/fill_rect  # run a single test case
```

`go generate` requires Ghostscript (`gs`).

## Test Infrastructure

Tests compare rendered output against Ghostscript-rendered reference images in
`testdata/reference/`.  The pass/fail tolerance is defined by the percentile
thresholds in `compareImages` (raster_test.go).

On failure, a 3-panel image is written to `debug/`: actual output (left),
diff (middle), reference (right).  In the diff panel green marks
under-production (expected > actual) and red marks over-production
(expected < actual).

To add a test case, add it to the appropriate file in `testcases/`, run
`go generate` to regenerate the references, then `go test`.
