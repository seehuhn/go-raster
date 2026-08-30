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
	"math"

	"seehuhn.de/go/geom/vec"
)

// edge represents a line segment in device coordinates.
type edge struct {
	x          float64 // x at ymin
	ymin, ymax float64 // vertical extent
	dxdy       float64 // dx/dy, precomputed for x-intercept calculation
	sign       float32 // +1 if the edge runs downward (y1 > y0), else -1
}

// addEdge transforms an edge from user space to device space, clips it to
// the clip rectangle, and appends the result to the edge list.
//
// Clipping bounds every later stage by the clip size, whatever the input
// coordinates: the edge is trimmed to the clip's vertical extent, the part
// to the right of the clip is dropped (it contributes to no pixel inside
// the clip), and the part to the left is replaced by a vertical edge one
// pixel left of the clip, which carries the same winding contribution.
func (r *Rasterizer) addEdge(p0, p1 vec.Vec2) {
	x0 := r.CTM[0]*p0.X + r.CTM[2]*p0.Y + r.CTM[4]
	y0 := r.CTM[1]*p0.X + r.CTM[3]*p0.Y + r.CTM[5]
	x1 := r.CTM[0]*p1.X + r.CTM[2]*p1.Y + r.CTM[4]
	y1 := r.CTM[1]*p1.X + r.CTM[3]*p1.Y + r.CTM[5]

	// a non-finite CTM or coordinate leaves no meaningful edge
	if !(math.Abs(x0) <= math.MaxFloat64 && math.Abs(y0) <= math.MaxFloat64 &&
		math.Abs(x1) <= math.MaxFloat64 && math.Abs(y1) <= math.MaxFloat64) {
		return
	}

	dy := y1 - y0
	if dy > -horizontalEdgeThreshold && dy < horizontalEdgeThreshold {
		return
	}
	dxdy := (x1 - x0) / dy
	if !(math.Abs(dxdy) <= math.MaxFloat64) {
		// The x span overflows float64 relative to dy, so across the clip's
		// width the edge rises by far less than one ulp of y: it is
		// horizontal wherever it can contribute. Dropping it here also keeps
		// the infinity out of the clipping arithmetic below.
		return
	}

	// trim to the clip's vertical extent
	yLo, yHi := float64(r.Clip.Min.Y), float64(r.Clip.Max.Y)
	if dy > 0 {
		if y1 <= yLo || y0 >= yHi {
			return
		}
		if y0 < yLo {
			x0 += dxdy * (yLo - y0)
			y0 = yLo
		}
		if y1 > yHi {
			x1 += dxdy * (yHi - y1)
			y1 = yHi
		}
	} else {
		if y0 <= yLo || y1 >= yHi {
			return
		}
		if y1 < yLo {
			x1 += dxdy * (yLo - y1)
			y1 = yLo
		}
		if y0 > yHi {
			x0 += dxdy * (yHi - y0)
			y0 = yHi
		}
	}

	// clip horizontally
	xLo, xHi := float64(r.Clip.Min.X), float64(r.Clip.Max.X)
	xLeft := xLo - 1
	if min(x0, x1) >= xHi {
		// The edge still bounds the filled region on the right, so it
		// must count towards the bounding box even though it is dropped.
		r.growBBox(xHi, xHi, y0, y1)
		return
	}
	if max(x0, x1) < xLo {
		r.appendEdge(xLeft, y0, xLeft, y1)
		return
	}
	if min(x0, x1) < xLo {
		// split where the edge enters the clip; dxdy is non-zero here
		ySplit := y0 + (xLo-x0)/dxdy
		if x0 < xLo {
			r.appendEdge(xLeft, y0, xLeft, ySplit)
			x0, y0 = xLo, ySplit
		} else {
			r.appendEdge(xLeft, ySplit, xLeft, y1)
			x1, y1 = xLo, ySplit
		}
	}
	if max(x0, x1) > xHi {
		// drop the part to the right of the clip
		ySplit := y0 + (xHi-x0)/dxdy
		if x0 > xHi {
			x0, y0 = xHi, ySplit
		} else {
			x1, y1 = xHi, ySplit
		}
	}
	r.appendEdge(x0, y0, x1, y1)
}

// appendEdge appends an edge in device coordinates, skipping horizontal
// edges and edges with a non-finite vertical extent, and grows the bounding
// box to include it.
func (r *Rasterizer) appendEdge(x0, y0, x1, y1 float64) {
	dy := y1 - y0
	if dy > -horizontalEdgeThreshold && dy < horizontalEdgeThreshold {
		return
	}
	if !(math.Abs(dy) <= math.MaxFloat64) {
		// a split point in addEdge overflowed; no usable edge is left
		return
	}
	x, ymin, sign := x0, y0, float32(1)
	if dy < 0 {
		x, ymin, sign = x1, y1, -1
	}
	r.edges = append(r.edges, edge{
		x:    x,
		ymin: ymin,
		ymax: max(y0, y1),
		dxdy: (x1 - x0) / dy,
		sign: sign,
	})
	r.growBBox(x0, x1, y0, y1)
}

// growBBox extends the device-space bounding box to include the given
// coordinates.
func (r *Rasterizer) growBBox(x0, x1, y0, y1 float64) {
	if r.edgeBBoxFirst {
		r.edgeDevXMin = min(x0, x1)
		r.edgeDevXMax = max(x0, x1)
		r.edgeDevYMin = min(y0, y1)
		r.edgeDevYMax = max(y0, y1)
		r.edgeBBoxFirst = false
	} else {
		r.edgeDevXMin = min(r.edgeDevXMin, x0, x1)
		r.edgeDevXMax = max(r.edgeDevXMax, x0, x1)
		r.edgeDevYMin = min(r.edgeDevYMin, y0, y1)
		r.edgeDevYMax = max(r.edgeDevYMax, y0, y1)
	}
}

// Coverage accumulation model:
//
// For each pixel, we track two values:
//   cover: signed vertical extent of edges crossing this pixel column
//   area:  horizontal position weighting (how far right the crossing is)
//
// An edge crossing a pixel contributes:
//   cover = sign * dy   (where sign is +1 for downward, -1 for upward)
//   area  = cover * (1 - xFrac)   (where xFrac is the horizontal position within the pixel)
//
// Final coverage is computed by integrateScanline:
//   pixel_coverage = accumulated_cover + area[i]
//   accumulated_cover += cover[i]   (carry forward for next pixel)
//
// This computes the signed area of the path within each pixel, which gives
// anti-aliased coverage values when clamped to [0,1] (nonzero) or folded (even-odd).

// accumulateEdge adds a single edge's contribution to the cover and area buffers.
// The buffers are indexed by (x - bboxXMin), where bboxXMin/bboxXMax define the buffer range.
// For edges spanning multiple pixels horizontally, this function splits the edge at pixel
// boundaries and computes separate contributions for each pixel crossed.
//
// The return values are the first and last buffer index written. They are
// meaningful only if lo <= hi; the edge contributes nothing to this scanline
// otherwise, and callers must not fold lo into a running minimum without
// checking.
func (r *Rasterizer) accumulateEdge(e *edge, y int, cover, area []float32, bboxXMin, bboxXMax int) (lo, hi int) {
	// Compute the portion of the edge within this scanline [y, y+1)
	yTop := float64(y)
	yBot := float64(y + 1)

	// Clamp to edge's actual y extent
	yTop = max(yTop, e.ymin)
	yBot = min(yBot, e.ymax)

	if yBot <= yTop {
		return 0, -1
	}

	sign := e.sign

	// Compute x at the y boundaries of the edge segment within this scanline
	xAtYTop := e.x + e.dxdy*(yTop-e.ymin)
	xAtYBot := e.x + e.dxdy*(yBot-e.ymin)

	// Determine pixel range the edge spans (ensure left <= right for iteration)
	xLeft, xRight := xAtYTop, xAtYBot
	if xLeft > xRight {
		xLeft, xRight = xRight, xLeft
	}

	pixLeft := int(math.Floor(xLeft))
	pixRight := int(math.Floor(xRight))

	// Handle edge entirely to the left of bbox
	if pixRight < bboxXMin {
		coverVal := sign * float32(yBot-yTop)
		cover[0] += coverVal
		area[0] += coverVal
		return 0, 0
	}

	// Handle edge entirely to the right of bbox
	if pixLeft >= bboxXMax {
		return 0, -1
	}

	// For vertical edges or edges within a single pixel column
	if pixLeft == pixRight {
		return r.accumulateEdgeInColumn(e, yTop, yBot, sign, pixLeft, cover, area, bboxXMin, bboxXMax)
	}

	// Edge spans multiple pixels - process each pixel column in x-order
	// For each pixel, compute the y-extent of the edge within that column
	dydx := 1 / e.dxdy
	lo = max(pixLeft, bboxXMin) - bboxXMin
	hi = min(pixRight, bboxXMax-1) - bboxXMin

	for pix := pixLeft; pix <= pixRight; pix++ {
		// Compute y at column boundaries
		yAtPixLeft := e.ymin + dydx*(float64(pix)-e.x)
		yAtPixRight := e.ymin + dydx*(float64(pix+1)-e.x)

		// Clamp to edge's y-extent within scanline
		segYMin := max(min(yAtPixLeft, yAtPixRight), yTop)
		segYMax := min(max(yAtPixLeft, yAtPixRight), yBot)

		segDy := segYMax - segYMin
		if segDy <= 0 {
			continue
		}

		// Compute contribution for this segment
		coverVal := sign * float32(segDy)

		// Compute average x within this pixel column
		yMid := (segYMin + segYMax) / 2
		xMid := e.x + e.dxdy*(yMid-e.ymin)
		addPixelContribution(cover, area, pix, bboxXMin, bboxXMax, coverVal, xMid)
	}
	return lo, hi
}

// accumulateEdgeInColumn handles an edge segment that falls within a single
// pixel column.  It returns the buffer index written as both lo and hi, or
// lo > hi if nothing was written.
func (r *Rasterizer) accumulateEdgeInColumn(e *edge, yTop, yBot float64, sign float32, pix int, cover, area []float32, bboxXMin, bboxXMax int) (lo, hi int) {
	coverVal := sign * float32(yBot-yTop)

	if pix < bboxXMin {
		addPixelContribution(cover, area, pix, bboxXMin, bboxXMax, coverVal, 0)
		return 0, 0
	}
	if pix >= bboxXMax {
		return 0, -1
	}

	// average x within this pixel
	yMid := (yTop + yBot) / 2
	xMid := e.x + e.dxdy*(yMid-e.ymin)
	addPixelContribution(cover, area, pix, bboxXMin, bboxXMax, coverVal, xMid)

	idx := pix - bboxXMin
	return idx, idx
}

// addPixelContribution adds a segment's cover/area contribution to pixel
// column pix, clamping to the buffer range [bboxXMin, bboxXMax).  xMid is
// the edge's x position at the segment's mid-y, used to weight area by how
// far right the crossing falls within the pixel.
func addPixelContribution(cover, area []float32, pix, bboxXMin, bboxXMax int, coverVal float32, xMid float64) {
	if pix < bboxXMin {
		cover[0] += coverVal
		area[0] += coverVal
		return
	}
	if pix >= bboxXMax {
		return
	}
	xFrac := xMid - float64(pix)
	areaVal := coverVal * float32(1-xFrac)
	idx := pix - bboxXMin
	cover[idx] += coverVal
	area[idx] += areaVal
}
