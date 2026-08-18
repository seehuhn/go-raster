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

// Command genpdf generates reference images for raster tests.
// It creates PDFs from test cases and renders them to PNGs using Ghostscript.
package main

import (
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/document"
	"seehuhn.de/go/pdf/graphics/color"
	"seehuhn.de/go/raster/testcases"
)

const refDir = "testdata/reference"

func main() {
	// Create output directory
	if err := os.MkdirAll(refDir, 0755); err != nil {
		panic(err)
	}

	// Process all test cases
	for _, category := range slices.Sorted(maps.Keys(testcases.All)) {
		for _, tc := range testcases.All[category] {
			name := category + "_" + tc.Name
			pdfPath := filepath.Join(refDir, name+".pdf")
			pngPath := filepath.Join(refDir, name+".png")

			if err := generatePDF(tc, pdfPath); err != nil {
				panic(fmt.Errorf("%s: %w", name, err))
			}

			if err := renderPNG(pdfPath, pngPath); err != nil {
				panic(fmt.Errorf("%s: %w", name, err))
			}

			if err := stripPNGMetadata(pngPath); err != nil {
				panic(fmt.Errorf("%s: %w", name, err))
			}
		}
	}
}

func generatePDF(tc testcases.TestCase, pdfPath string) error {
	// Page size in points (1 point = 1 pixel at 72 DPI)
	paper := &pdf.Rectangle{
		URx: float64(tc.Width),
		URy: float64(tc.Height),
	}

	page, err := document.CreateSinglePage(pdfPath, paper, pdf.V1_7, nil)
	if err != nil {
		return err
	}

	// Paint black background first (PDF default is white, but we need
	// black background for coverage semantics: 0=no coverage, 255=full)
	page.SetFillColor(color.DeviceGray(0))
	page.Rectangle(0, 0, float64(tc.Width), float64(tc.Height))
	page.Fill()

	// PDF origin is bottom-left; test cases assume top-left.
	// Apply Y-axis flip.
	page.Transform(matrix.Matrix{1, 0, 0, -1, 0, float64(tc.Height)})

	// Apply test case CTM if present
	if tc.CTM != (matrix.Matrix{}) && tc.CTM != matrix.Identity {
		page.Transform(tc.CTM)
	}

	// Set white color for fill/stroke (on black background = coverage values)
	page.SetFillColor(color.DeviceGray(1))
	page.SetStrokeColor(color.DeviceGray(1))

	// Set stroke parameters before path construction (PDF requirement)
	if op, ok := tc.Op.(testcases.Stroke); ok {
		page.SetLineWidth(op.Width)
		page.SetLineCap(op.Cap)
		page.SetLineJoin(op.Join)
		page.SetMiterLimit(op.MiterLimit)
		if len(op.Dash) > 0 {
			page.SetLineDash(op.Dash, op.DashPhase)
		}
	}

	// Draw path - convert quadratic to cubic (PDF doesn't support quadratic)
	for cmd, pts := range tc.Path.Iter().ToCubic() {
		switch cmd {
		case path.CmdMoveTo:
			page.MoveTo(pts[0].X, pts[0].Y)
		case path.CmdLineTo:
			page.LineTo(pts[0].X, pts[0].Y)
		case path.CmdCubeTo:
			page.CurveTo(pts[0].X, pts[0].Y, pts[1].X, pts[1].Y, pts[2].X, pts[2].Y)
		case path.CmdClose:
			page.ClosePath()
		}
	}

	// Apply paint operation
	switch op := tc.Op.(type) {
	case testcases.Fill:
		if op.Rule == testcases.EvenOdd {
			page.FillEvenOdd()
		} else {
			page.Fill()
		}
	case testcases.Stroke:
		page.Stroke()
	}

	return page.Close()
}

func renderPNG(pdfPath, pngPath string) error {
	// Render PDF to PNG using Ghostscript
	// -sDEVICE=pnggray: 8-bit grayscale (matches Cairo FORMAT_A8)
	// -r72: 72 DPI (1 point = 1 pixel)
	// -dGraphicsAlphaBits=4: 4x supersampling for anti-aliasing
	cmd := exec.Command(
		"gs", "-q",
		"-sDEVICE=pnggray",
		"-r72",
		"-dGraphicsAlphaBits=4",
		"-o", pngPath,
		pdfPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// strippedChunks lists the PNG chunk types removed from reference images.
// None of them affect the pixel values that image/png decodes, but all are
// written by Ghostscript and would make regenerated references differ
// byte-wise even when the pixels are unchanged: the text chunks (tEXt,
// zTXt, iTXt) carry the Ghostscript version, tIME the generation time, and
// iCCP the bundled colour profile, which a Ghostscript upgrade may replace
// (2.3 kB in every file, roughly 95% of the total).  pHYs merely echoes the
// -r72 rendering resolution; the tests compare pixels, so it goes too.
var strippedChunks = map[string]bool{
	"tEXt": true,
	"zTXt": true,
	"iTXt": true,
	"tIME": true,
	"iCCP": true,
	"pHYs": true,
}

// stripPNGMetadata removes the chunks listed in strippedChunks from a PNG
// file.  The file is left untouched if it contains none of them.
func stripPNGMetadata(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	const signature = "\x89PNG\r\n\x1a\n"
	if len(data) < len(signature) || string(data[:len(signature)]) != signature {
		return fmt.Errorf("%s: not a PNG file", path)
	}

	out := make([]byte, 0, len(data))
	out = append(out, data[:len(signature)]...)
	pos := len(signature)
	for pos < len(data) {
		if pos+8 > len(data) {
			return fmt.Errorf("%s: truncated PNG chunk", path)
		}
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		end := pos + 12 + length
		if end < pos || end > len(data) {
			return fmt.Errorf("%s: truncated PNG chunk", path)
		}
		if !strippedChunks[string(data[pos+4:pos+8])] {
			out = append(out, data[pos:end]...)
		}
		pos = end
	}

	if len(out) == len(data) {
		return nil
	}
	return os.WriteFile(path, out, 0666)
}
