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

package raster

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/raster/testcases"
)

func TestAgainstReference(t *testing.T) {
	// Test each case with both approaches:
	// - Approach A (2D buffers): threshold = MaxInt (always use A)
	// - Approach B (active edge list): threshold = 0 (always use B)
	approaches := []struct {
		name      string
		threshold int
	}{
		{"A", 1 << 30}, // very large threshold forces Approach A
		{"B", 0},       // zero threshold forces Approach B
	}

	for _, category := range slices.Sorted(maps.Keys(testcases.All)) {
		for _, tc := range testcases.All[category] {
			baseName := category + "_" + tc.Name
			for _, approach := range approaches {
				name := baseName + "_" + approach.name
				threshold := approach.threshold
				t.Run(name, func(t *testing.T) {
					// load reference image
					refPath := filepath.Join("testdata", "reference", baseName+".png")
					ref, err := loadGray(refPath)
					if err != nil {
						t.Fatalf("loading reference: %v", err)
					}

					// allocate output buffer
					w, h := tc.Width, tc.Height
					actual := make([]byte, w*h)

					// render with specified approach threshold
					renderExample(tc, actual, w, h, w, threshold)

					// compare
					if err := compareImages(name, ref, actual, w, h); err != nil {
						t.Error(err)
					}
				})
			}
		}
	}
}

// renderExample renders a test case into a grayscale buffer.
// The buffer is pre-initialized with zeros, in row-major order.
// Each byte represents coverage from 0 (transparent) to 255 (opaque).
// The threshold parameter controls the Approach A/B cutoff for testing.
func renderExample(tc testcases.TestCase, buf []byte, width, height, stride int, threshold int) {
	clip := rect.Rect{
		LLx: 0,
		LLy: 0,
		URx: float64(width),
		URy: float64(height),
	}
	r := NewRasterizer(clip)
	r.smallPathThreshold = threshold

	// Apply CTM (zero-value means identity, which is already the default)
	if tc.CTM != (matrix.Matrix{}) {
		r.CTM = tc.CTM
	}

	// Emit callback: convert float32 coverage to bytes
	emit := func(y, xMin int, coverage []float32) {
		row := buf[y*stride:]
		for i, c := range coverage {
			row[xMin+i] = byte(max(0, min(255, int(c*256))))
		}
	}

	// Dispatch based on operation type
	switch op := tc.Op.(type) {
	case testcases.Fill:
		if op.Rule == testcases.EvenOdd {
			r.FillEvenOdd(tc.Path.Iter(), emit)
		} else {
			r.FillNonZero(tc.Path.Iter(), emit)
		}
	case testcases.Stroke:
		r.Width = op.Width
		r.Cap = op.Cap
		r.Join = op.Join
		r.MiterLimit = op.MiterLimit
		r.Dash = op.Dash
		r.DashPhase = op.DashPhase
		r.Stroke(tc.Path.Iter(), emit)
	}
}

func loadGray(path string) (gray []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	gray = make([]byte, w*h)

	for y := range h {
		for x := range w {
			c := color.GrayModel.Convert(img.At(x+bounds.Min.X, y+bounds.Min.Y)).(color.Gray)
			gray[y*w+x] = c.Y
		}
	}
	return gray, nil
}

func compareImages(name string, expected, actual []byte, w, h int) error {
	total := w * h

	// Collect all absolute differences
	diffs := make([]int, total)
	for i := range total {
		e, a := int(expected[i]), int(actual[i])
		diff := e - a
		if diff < 0 {
			diff = -diff
		}
		diffs[i] = diff
	}

	// Sort differences to compute percentiles
	sort.Ints(diffs)

	// Compute percentile values
	p80 := diffs[int(math.Round(0.80*float64(total-1)))]
	p95 := diffs[int(math.Round(0.95*float64(total-1)))]
	p99 := diffs[int(math.Round(0.99*float64(total-1)))]

	// Check criteria (thresholds set just above the observed worst case
	// across all reference images, leaving headroom for cross-platform
	// floating-point drift):
	// - at least 80% of pixels are identical (p80 == 0)
	// - at least 95% of differences are < 48 (p95 < 48)
	// - at least 99% of differences are < 96 (p99 < 96)
	var failures []string
	if p80 > 0 {
		failures = append(failures, fmt.Sprintf("80th percentile diff is %d (want 0)", p80))
	}
	if p95 >= 48 {
		failures = append(failures, fmt.Sprintf("95th percentile diff is %d (want <48)", p95))
	}
	if p99 >= 96 {
		failures = append(failures, fmt.Sprintf("99th percentile diff is %d (want <96)", p99))
	}

	if len(failures) > 0 {
		_ = writeDiffImage(name, expected, actual, w, h)
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func writeDiffImage(name string, expected, actual []byte, w, h int) (err error) {
	if err := os.MkdirAll("debug", 0755); err != nil {
		return err
	}

	// Create 3-panel image: actual (left), diff (middle), reference (right)
	img := image.NewRGBA(image.Rect(0, 0, w*3, h))
	for y := range h {
		for x := range w {
			i := y*w + x

			// Left panel: actual output (grayscale)
			a := actual[i]
			img.Set(x, y, color.RGBA{R: a, G: a, B: a, A: 255})

			// Middle panel: diff (green=under, red=over, black=match)
			diff := int(expected[i]) - int(actual[i])
			var diffColor color.RGBA
			if diff > 0 {
				// Under-producing (expected > actual): green
				diffColor = color.RGBA{R: 0, G: uint8(diff), B: 0, A: 255}
			} else if diff < 0 {
				// Over-producing (expected < actual): red
				diffColor = color.RGBA{R: uint8(-diff), G: 0, B: 0, A: 255}
			} else {
				// No difference: black
				diffColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
			}
			img.Set(x+w, y, diffColor)

			// Right panel: reference/expected (grayscale)
			e := expected[i]
			img.Set(x+w*2, y, color.RGBA{R: e, G: e, B: e, A: 255})
		}
	}

	f, err := os.Create(filepath.Join("debug", name+".png"))
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// TestTriangleCoverage verifies exact coverage values for a simple triangle.
// The triangle (0,0)→(10,0)→(10,1)→close has a diagonal edge y = x/10.
// Each pixel X should have coverage (2X+1)/20: 0.05, 0.15, ..., 0.95.
func TestTriangleCoverage(t *testing.T) {
	// Build the triangle path in device space
	trianglePath := (&path.Data{}).
		MoveTo(vec.Vec2{X: 0, Y: 0}).
		LineTo(vec.Vec2{X: 10, Y: 0}).
		LineTo(vec.Vec2{X: 10, Y: 1}).
		Close()

	// Create rasterizer with clip covering the triangle
	clip := rect.Rect{LLx: 0, LLy: 0, URx: 10, URy: 1}
	r := NewRasterizer(clip)

	// Collect coverage values
	coverage := make([]float32, 10)
	emit := func(y, xMin int, cov []float32) {
		if y == 0 {
			for i, c := range cov {
				coverage[xMin+i] = c
			}
		}
	}

	r.FillNonZero(trianglePath.Iter(), emit)

	// Verify each pixel's coverage
	const epsilon = 1e-6
	for x := range 10 {
		expected := float32(2*x+1) / 20.0 // 0.05, 0.15, ..., 0.95
		actual := coverage[x]
		if math.Abs(float64(actual-expected)) > epsilon {
			t.Errorf("pixel %d: expected coverage %.4f, got %.4f", x, expected, actual)
		}
	}
}

// BenchmarkRasterizeAll measures steady-state performance by reusing a single
// Rasterizer across all test cases. This tests buffer reuse with varying clip sizes.
func BenchmarkRasterizeAll(b *testing.B) {
	// Collect all test cases
	var cases []testcases.TestCase
	for _, category := range slices.Sorted(maps.Keys(testcases.All)) {
		cases = append(cases, testcases.All[category]...)
	}

	// Create rasterizer once, reuse across all iterations
	r := NewRasterizer(rect.Rect{})

	// No-op emit callback - we're measuring rasterization, not compositing
	emit := func(y, xMin int, coverage []float32) {}

	b.ResetTimer()
	for b.Loop() {
		for _, tc := range cases {
			// Update clip for this test case
			r.Clip = rect.Rect{
				LLx: 0,
				LLy: 0,
				URx: float64(tc.Width),
				URy: float64(tc.Height),
			}

			// Apply CTM
			if tc.CTM != (matrix.Matrix{}) {
				r.CTM = tc.CTM
			} else {
				r.CTM = matrix.Identity
			}

			// Render
			switch op := tc.Op.(type) {
			case testcases.Fill:
				if op.Rule == testcases.EvenOdd {
					r.FillEvenOdd(tc.Path.Iter(), emit)
				} else {
					r.FillNonZero(tc.Path.Iter(), emit)
				}
			case testcases.Stroke:
				r.Width = op.Width
				r.Cap = op.Cap
				r.Join = op.Join
				r.MiterLimit = op.MiterLimit
				r.Dash = op.Dash
				r.DashPhase = op.DashPhase
				r.Stroke(tc.Path.Iter(), emit)
			}
		}
	}
}
