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

package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeChunk assembles a PNG chunk with the given type and body.
func makeChunk(ctype string, body []byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(body)))
	buf.WriteString(ctype)
	buf.Write(body)
	crc := crc32.NewIEEE()
	crc.Write([]byte(ctype))
	crc.Write(body)
	binary.Write(&buf, binary.BigEndian, crc.Sum32())
	return buf.Bytes()
}

// chunkTypes lists the chunk types of a PNG file in order.
func chunkTypes(t *testing.T, data []byte) []string {
	t.Helper()
	var types []string
	pos := 8
	for pos < len(data) {
		if pos+8 > len(data) {
			t.Fatal("truncated chunk header")
		}
		length := binary.BigEndian.Uint32(data[pos : pos+4])
		types = append(types, string(data[pos+4:pos+8]))
		pos += 12 + int(length)
	}
	if pos != len(data) {
		t.Fatal("trailing garbage after last chunk")
	}
	return types
}

func TestStripPNGMetadata(t *testing.T) {
	// a small image, encoded and then decorated with metadata chunks
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Pix[0] = 200
	var enc bytes.Buffer
	if err := png.Encode(&enc, img); err != nil {
		t.Fatal(err)
	}
	plain := enc.Bytes()

	ihdrEnd := 8 + 12 + 13 // signature + IHDR chunk
	var dec bytes.Buffer
	dec.Write(plain[:ihdrEnd])
	dec.Write(makeChunk("tEXt", []byte("Software\x00GPL Ghostscript 10.07.1")))
	dec.Write(makeChunk("tIME", []byte{7, 234, 8, 15, 12, 0, 0}))
	dec.Write(plain[ihdrEnd:])

	path := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(path, dec.Bytes(), 0666); err != nil {
		t.Fatal(err)
	}

	if err := stripPNGMetadata(path); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, ctype := range chunkTypes(t, got) {
		switch ctype {
		case "tEXt", "zTXt", "iTXt", "tIME":
			t.Errorf("metadata chunk %q not stripped", ctype)
		}
	}

	// the image data must be untouched
	gotImg, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	wantImg, err := png.Decode(bytes.NewReader(plain))
	if err != nil {
		t.Fatal(err)
	}
	b := wantImg.Bounds()
	if gotImg.Bounds() != b {
		t.Fatalf("bounds changed: %v != %v", gotImg.Bounds(), b)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if gotImg.At(x, y) != wantImg.At(x, y) {
				t.Fatalf("pixel (%d,%d) changed", x, y)
			}
		}
	}
}

func TestStripPNGMetadataNoChange(t *testing.T) {
	// a file without metadata chunks must not be rewritten
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var enc bytes.Buffer
	if err := png.Encode(&enc, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(path, enc.Bytes(), 0666); err != nil {
		t.Fatal(err)
	}

	// Backdate the file, so that a rewrite is detected regardless of the
	// timestamp granularity of the underlying file system.
	before := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, before, before); err != nil {
		t.Fatal(err)
	}

	if err := stripPNGMetadata(path); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(before) {
		t.Error("file without metadata was rewritten")
	}
}

func TestStripPNGMetadataMalformed(t *testing.T) {
	const signature = "\x89PNG\r\n\x1a\n"
	ihdr := string(makeChunk("IHDR", make([]byte, 13)))

	cases := []struct {
		name string
		data []byte
	}{
		{"empty file", nil},
		{"wrong signature", []byte("not a png")},
		{"truncated chunk header", []byte(signature + ihdr + "\x00\x00\x00")},
		{"chunk extends past end", []byte(signature + ihdr + "\x00\x00\x10\x00tEXt")},
		{"chunk length overflows int", []byte(signature + ihdr + "\xff\xff\xff\xfftEXt")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.png")
			if err := os.WriteFile(path, tc.data, 0666); err != nil {
				t.Fatal(err)
			}
			if err := stripPNGMetadata(path); err == nil {
				t.Error("no error for malformed file")
			}
		})
	}
}

func TestStripPNGMetadataMissing(t *testing.T) {
	if err := stripPNGMetadata(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Error("no error for missing file")
	}
}
