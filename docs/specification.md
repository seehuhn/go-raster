# 2D Vector Graphics Rasteriser Specification

## 1. Scope and Goals

This document specifies a rasteriser for the PDF/PostScript imaging model. The rasteriser produces anti-aliased coverage values without supersampling, suitable for page graphics and font glyphs.

Input paths comprise straight line segments, quadratic Bézier curves, and cubic Bézier curves. Coordinates are floating-point in user space, yielding sub-pixel precision after transformation. The fill rule may be nonzero winding or even-odd. Strokes take additional parameters: line width, cap style, join style, miter limit, and dash pattern.

Output is per-pixel coverage in the range [0.0, 1.0], representing the fraction of each pixel covered by the path interior. The rasteriser delivers output row-by-row to a compositor.

---

## 2. Coordinate Systems and Renderer State

This section defines the coordinate systems and state variables used throughout. The coordinate systems determine how paths transform before rasterisation (§3). The state variables control both rasterisation and stroke expansion (§6).

### 2.1 Coordinate Systems

The renderer operates in two coordinate systems.

User space is where paths are defined—the coordinate system of the PDF/PostScript content stream. Stroke parameters (line width, dash lengths) use user-space units.

Device space is the output pixel grid. One unit equals one pixel. The coverage accumulation algorithm (§3) operates here.

### 2.2 Current Transformation Matrix (CTM)

The CTM transforms points from user space to device space. In matrix form,
```
[x_device]   [a  b  tx] [x_user]
[y_device] = [c  d  ty] [y_user]
[   1    ]   [0  0   1] [   1   ]
```
which expands to
```
x_device = a × x_user + b × y_user + tx
y_device = c × x_user + d × y_user + ty
```

The 2×2 linear part M = [a b; c d] controls scaling, rotation, and shear. The translation (tx, ty) controls position.

A CTM is conformal if it preserves angles—that is, if it consists only of uniform scaling, rotation, and translation. Conformality requires a = d, b = −c, and a² + b² > 0. Under a conformal CTM, circles remain circles. A non-conformal CTM includes non-uniform scaling or shear, turning circles into ellipses.

### 2.3 Renderer State

The renderer maintains the current transformation matrix (user space to device space), the flatness tolerance (maximum deviation in device pixels), and the fill rule (nonzero winding or even-odd).

For stroke operations, the renderer also maintains the stroke width in user-space units, the cap style (butt, round, or square), the join style (miter, round, or bevel), the miter limit as a dimensionless ratio, the dash pattern as an array of dash/gap lengths in user-space units, and the dash phase as an offset into the pattern.

### 2.4 Processing Pipelines

Fill and stroke operations share a common final stage but differ in preparation.

The fill pipeline receives the path in user space, flattens curves (§5), transforms the flattened path to device space, and rasterises the line segments (§3).

The stroke pipeline receives the path in user space, flattens curves (§5), expands to a stroke outline in user space (§6), transforms the outline to device space, and rasterises the line segments (§3).

Both pipelines end by transforming to device space and rasterising. The core rasteriser (§3) always operates on line segments in device coordinates.

### 2.5 Subpath Closing

Each subpath is implicitly closed. When a MoveTo command starts a new subpath, the rasteriser adds a closing edge from the current point back to the start of the previous subpath (if any). After processing all path commands, the final subpath is also closed. This matches PDF/PostScript semantics where all filled regions are closed shapes.

Explicit Close commands have the same effect as an implicit close followed by setting the current point to the subpath start. A path with only MoveTo commands (no drawing operations) produces no edges.

---

## 3. Core Algorithm: Signed-Area Coverage Accumulation

This section describes how line segments become pixel coverage values. All paths—fills and stroke outlines alike—pass through this algorithm after transformation to device space. The algorithm computes exact fractional coverage without supersampling.

### 3.1 Principle

Each directed edge contributes a signed trapezoidal area extending horizontally to the right. For each pixel an edge touches, the algorithm computes two values. The cover value is the change in winding number as we cross the pixel vertically. The area value is the signed area within that pixel. These values accumulate in row buffers, then integrate left-to-right to produce final coverage.

### 3.2 Edge Contribution

Consider a directed segment from P0 = (x0, y0) to P1 = (x1, y1) crossing a pixel at integer coordinates (X, Y). The pixel spans [X, X+1) horizontally and [Y, Y+1) vertically.

Horizontal edges (y0 = y1) have no vertical extent and contribute nothing; skip them.

For other edges, compute the portion within the pixel. Let y_top = max(Y, min(y0, y1)) and y_bot = min(Y+1, max(y0, y1)), so dy = y_bot − y_top is always non-negative.

The cover contribution is the signed vertical extent:
```
cover = (y_bot - y_top) × sign(y1 - y0)
```
This equals +dy for downward edges and −dy for upward edges.

The area contribution accounts for the edge's horizontal position. If the edge crosses the pixel at x_mid (the x-coordinate at the vertical midpoint within this pixel), then
```
area = cover × (1.0 - (x_mid - X))
```
where x_mid − X is the fractional x-position (0.0 at left, 1.0 at right).

For steep edges, x_mid may fall outside [X, X+1). This is correct; do not clamp. Integration across adjacent pixels compensates for area values outside [0, cover].

When an edge lies entirely left of the rendering region (x_mid < X, where X is the leftmost pixel column), it still contributes to winding for all pixels to its right. For such edges, set area = cover. Using the formula with a large negative x-fraction would produce incorrect values. An edge entirely left of a pixel has already fully entered by the time integration reaches that pixel, so its entire winding contribution applies.

For non-vertical edges, compute x_mid by linear interpolation. Let y_mid = (y_top + y_bot) / 2, giving
```
x_mid = x0 + (x1 - x0) × (y_mid - y0) / (y1 - y0)
```
Since horizontal edges were excluded, y1 − y0 is nonzero and this division is safe.

For a vertical edge at x = x0, the area simplifies to cover × (1.0 − frac(x0)), where frac(x0) = x0 − floor(x0).

### 3.3 Integration

After processing all edges for scanline Y, compute final coverage by integrating left-to-right:

```
accumulated_cover = 0.0
for x from x_min to x_max:
    raw_coverage = accumulated_cover + area[x]
    final_coverage[x] = apply_fill_rule(raw_coverage)
    accumulated_cover += cover[x]
```

### 3.4 Fill Rules

The fill rule converts raw coverage to final coverage.

For nonzero winding, apply_fill_rule(c) = clamp(abs(c), 0.0, 1.0).

For even-odd, apply_fill_rule(c) = 1.0 − abs(1.0 − mod(abs(c), 2.0)), where mod(x, 2.0) = x − 2.0 × floor(x / 2.0). This produces a sawtooth wave: as |c| rises from 0 to 1, coverage rises from 0 to 1; as |c| continues from 1 to 2, coverage falls from 1 to 0; the pattern repeats.

### 3.5 Edge Direction

Edge direction determines winding. Downward edges (y1 > y0) contribute positively to cover. Upward edges (y1 < y0) contribute negatively. Horizontal edges (y1 = y0) contribute nothing and are skipped, as noted in §3.2.

---

## 4. Buffer Management

This section describes two approaches for managing coverage accumulation buffers. The choice affects memory and complexity but not output. Implementations may use either or both, selected by heuristic.

### 4.1 Approach A: 2D Buffers (Small Paths)

For small paths (glyphs, icons), allocate 2D buffers sized to the bounding box: cover[y][x] and area[y][x] as float32 arrays.

First, compute the bounding box in device coordinates and clamp to clip bounds. Second, allocate or reuse buffers. Third, process edges in path order; each edge writes to all scanlines it touches. Fourth, integrate and emit each row.

Memory usage is O(width × height). A 256×256 region requires roughly 512 KB. This approach needs no sorting and suits paths where width × height is small (under 256K pixels).

### 4.2 Approach B: 1D Buffers and Active Edge List (Large Paths)

For large paths (page-spanning fills), allocate 1D buffers one row tall: cover[x] and area[x] as float32 arrays.

First, compute the bounding box and clamp to clip bounds. Second, build an edge list storing (x0, y0, x1, y1) per edge. Third, sort edges by y_min. Fourth, process scanlines from y_min to y_max: add starting edges to the active list, compute contributions, remove ending edges, integrate and emit, then clear the buffers.

Memory usage is O(width + edges), independent of height. This approach requires sorting but suits large regions.

### 4.3 Selection Heuristic

Use Approach A when bounding box area (width × height in pixels) falls below a threshold; use Approach B otherwise. Glyphs use the simpler 2D approach; page-spanning fills use the active edge list. Tune the threshold by profiling.

### 4.4 Output

After integrating each scanline, pass coverage data to the compositor: the Y coordinate, x_min, x_max, and the coverage array.

---

## 5. Curve Flattening

This section describes how Bézier curves become line segments before rasterisation. Flattening precedes both fill and stroke operations (§2.4). The tolerance is in device pixels, ensuring consistent visual quality regardless of transformation.

### 5.1 Tolerance Checking

The flatness tolerance ε is in device pixels (typical values: 0.25 to 1.0). Since flattening occurs in user space but accuracy is measured in device space, deviation vectors must be transformed before comparison.

Compute the deviation vector in user space, transform it by the CTM's linear part M (ignoring translation), and compare the magnitude against ε. For a user-space deviation d_user, the transformed vector is d_device = M × d_user, and the curve is flat enough when ||d_device|| ≤ ε.

This approach generates the optimal segment count for any CTM, including non-conformal transformations.

### 5.2 Quadratic Bézier Curves

A quadratic Bézier with control points P0, P1, P2 flattens using Levien's formula.

The maximum error vector in user space is e_user = (P0 − 2×P1 + P2) / 4. Transforming gives e_device = M × e_user.

If ||e_device|| ≤ ε, draw a single line from P0 to P2.

Otherwise, subdividing into n equal parameter intervals divides the error by n². The segment count is therefore n = ceil(sqrt(||e_device|| / ε)).

For a conformal CTM with scale s, this simplifies to n = ceil(sqrt(s × ||e_user|| / ε)).

For a non-conformal CTM, compute ||e_device|| directly. Alternatively, use the maximum singular value σ_max of M as a conservative bound, giving n = ceil(sqrt(σ_max × ||e_user|| / ε)).

The maximum singular value is
```
σ_max = sqrt((a² + b² + c² + d²) / 2 + sqrt(((a² + b²) - (c² + d²))² / 4 + (a×c + b×d)²))
```
or conservatively σ_max ≤ sqrt(max(a² + b², c² + d²)) × sqrt(2).

Subdivide [0, 1] into n intervals and evaluate at t = i/n for i = 0, 1, ..., n using B(t) = (1−t)²×P0 + 2×(1−t)×t×P1 + t²×P2.

### 5.3 Cubic Bézier Curves

A cubic Bézier with control points P0, P1, P2, P3 flattens using Wang's formula, which computes the segment count directly without recursive subdivision.

The maximum deviation is bounded by the bend at each curve end. Define d1_user = P0 − 2×P1 + P2 and d2_user = P1 − 2×P2 + P3. These measure how far the control polygon deviates from collinearity.

Transform to device space: d1_device = M × d1_user and d2_device = M × d2_user. Let M_dev = max(||d1_device||, ||d2_device||).

If M_dev ≤ ε, one segment suffices. Otherwise, n = ceil(sqrt(3 × M_dev / (4 × ε))).

For a conformal CTM with scale s, let M_user = max(||d1_user||, ||d2_user||), giving n = ceil(sqrt(3 × s × M_user / (4 × ε))).

For a non-conformal CTM, compute the device-space norms directly, or use σ_max as a bound: n = ceil(sqrt(3 × σ_max × M_user / (4 × ε))).

Subdivide [0, 1] into n intervals and evaluate at t = i/n using B(t) = (1−t)³×P0 + 3×(1−t)²×t×P1 + 3×(1−t)×t²×P2 + t³×P3.

If all four control points coincide, the curve degenerates to a point. Emit a single segment from P0 to P3; it contributes nothing to coverage.

---

## 6. Stroke Expansion

This section describes how strokes become filled outlines. After expansion, the outline rasterises using the same algorithm as fills (§3). All stroke parameters use user-space units.

### 6.1 Overview

Stroke expansion proceeds as follows. First, flatten curves (§5). Second, apply the dash pattern, splitting the path at transitions. Third, emit pieces covering the segments, joins, and caps: runs of gently turning segments merge into strips (§6.11); every other segment becomes a quadrilateral (§6.2), with one piece per remaining join (§6.4) and per cap (§6.3). The result is a set of simple polygons in user space. The pipeline (§2.4) then transforms them to device space.

The stroked region is the union of these pieces. Every piece is simple and wound clockwise, so under the nonzero winding rule (§3) overlapping pieces reinforce one another and the union fills uniformly. Winding direction is therefore load-bearing: a piece wound the other way would cancel the pieces it overlaps and leave a hole.

Tracing a single contour around the outline instead does not work. Wherever a stroke covers the same area twice — a tight curve, a self-crossing path, a dash landing on an earlier one — the contour reverses orientation and the winding cancels to zero.

### 6.2 Line Segment Offsetting

Consider a segment from A to B. If ||B − A|| < δ for a small tolerance δ, the segment has no defined direction; skip it. (Zero-length subpaths are handled in §6.9.)

For other segments, compute the unit tangent T = normalise(B − A), the unit normal N = (−T.y, T.x), and the offset d = line_width / 2.

The segment contributes the quadrilateral A + d×N, B + d×N, B − d×N, A − d×N, in that order. This runs up the +N side and back down the −N side, which is clockwise.

### 6.3 Line Caps

Caps appear at the start and end of open subpaths and at dash endpoints. Each is a separate piece.

Let P be the endpoint, T the outward tangent (away from the path), N = (−T.y, T.x), and d = line_width / 2.

A butt cap adds no piece. The segment quadrilateral already ends flush at P.

A square cap adds the quadrilateral P + d×N, P + d×N + d×T, P − d×N + d×T, P − d×N.

A round cap adds a half-disc: the semicircular arc centred at P with radius d runs from P + d×N clockwise through P + d×T to P − d×N, and the diameter closes it. Approximate the arc with line segments (§6.6).

### 6.4 Line Joins

Joins appear where consecutive segments meet at point P, including at the closing corner of a closed subpath. Each is a separate piece, filling the wedge the two segment quadrilaterals leave open on the outside of the corner.

Let T1 be the incoming tangent, T2 the outgoing tangent, and d = line_width / 2. The signed turn angle is θ = atan2(cross(T1, T2), dot(T1, T2)), where the 2D cross product is T1.x × T2.y − T1.y × T2.x. It lies in (−180°, 180°], and is positive for a left turn.

Taking θ from atan2 rather than from the sign of the cross product alone keeps the cusp case well defined (§6.8): where the path doubles back the cross product vanishes, but atan2 still yields ±180° and picks out the side the path turns towards.

If |θ| < δ the path continues straight, the two quadrilaterals meet flush, and no piece is needed. A joint folded into a strip (§6.11) also gets no piece; there the strip's chord stands in for the join.

The join lies on the outer side of the corner, away from the turn. Let s = −sign(θ), and let the outer normals be n1 = s × (−T1.y, T1.x) and n2 = s × (−T2.y, T2.x). They span exactly the turn angle. Order them so that the sweep from the first to the second is clockwise: take (n1, n2) for a right turn and (n2, n1) for a left turn, and call the result n_from and n_to.

A bevel join adds the triangle P, P + d×n_from, P + d×n_to.

A round join adds the circular sector P, then the arc of radius d from P + d×n_from clockwise to P + d×n_to. Approximate the arc with segments (§6.6).

A miter join adds the quadrilateral P, P + d×n_from, tip, P + d×n_to, where the tip is P + normalise(n1 + n2) × (d / half_angle).

The miter length is the distance from inner edge to miter tip. Per PDF, miter_length / line_width = 1 / sin(φ/2), where φ is the corner's interior angle. Since |θ| is the angle between tangents, φ = 180° − |θ| and sin(φ/2) = cos(θ/2).

The half-angle identity gives cos(θ/2) = sqrt((1 + dot(T1, T2)) / 2). Define half_angle = sqrt((1 + dot(T1, T2)) / 2), so miter_length = line_width / half_angle.

If 1 / half_angle exceeds the miter_limit, use a bevel instead.

The inner side of the corner needs nothing. There the two segment quadrilaterals overlap, and the nonzero winding rule paints the overlap once (§6.10).

### 6.5 Miter Limit

The miter limit M is a dimensionless ratio, at least 1.0.

It corresponds to a minimum interior angle φ_min, where M = 1 / sin(φ_min / 2). For M = 10, φ_min ≈ 11.5°.

When φ < φ_min, a bevel replaces the miter.

### 6.6 Arc Approximation

Round caps and joins are circles in user space, becoming ellipses under non-conformal CTMs. This section explains how to choose segment counts for device-space accuracy.

Generate arc vertices in user space, but determine count by considering the arc's device-space appearance.

Let d = line_width / 2, M be the CTM's linear part, and ε the flatness tolerance.

For a conformal CTM with scale s, the circle remains circular with device radius s × d. The segment count is n = ceil(π / acos(1 − ε / (s × d))).

For a non-conformal CTM, the circle becomes an ellipse. Error is greatest at the major axis ends. Compute the semi-major axis as σ_max × d, where σ_max is M's maximum singular value (§5.2). The segment count is n = ceil(π / acos(1 − ε / (σ_max × d))).

For a full circle, generate n+1 vertices. For i from 0 to n, let angle = 2π × i / n and vertex[i] = centre + (d × cos(angle), d × sin(angle)).

For a semicircle from angle α to α + π, let segments = ceil(n / 2). For i from 0 to segments, let angle = α + π × i / segments.

For an arc spanning angle θ, let segments = ceil(n × θ / (2π)). For i from 0 to segments, let angle = start_angle + θ × i / segments.

### 6.7 Dash Patterns

A dash pattern [d0, d1, d2, ...] specifies alternating on/off lengths in user-space units. The elements must not all be zero. The dash phase offsets into the pattern.

If the array has odd length, double it first. Pattern [3] becomes [3, 3]; pattern [2, 1, 4] becomes [2, 1, 4, 2, 1, 4].

If the phase is negative, add twice the total pattern length until non-negative.

To apply the pattern, first compute total length L. Second, normalise the phase to [0, L). Third, walk the flattened path measuring arc length. Fourth, at each on/off transition, split the path. Each on-segment becomes an open subpath with caps at both ends.

Each subpath in a multi-subpath path is treated independently; the pattern restarts.

In a closed dashed subpath, if the first dash starts on and the last ends on, join them with a line join rather than caps.

If a corner falls within an on-segment, paint it with the join style. If within an off-segment, skip it. If a dash ends exactly at a corner, paint the cap before the join.

Arc length along a segment from A to B is ||B − A||. Summing segment lengths gives cumulative arc length. Since dashing follows flattening, this measures length along the polygonal approximation, closely matching true arc length (error is O(ε²)).

### 6.8 Cusps

A cusp occurs where a path doubles back on itself, making the turn angle 180° and laying the two segment quadrilaterals on top of each other.

No special case is needed. The join piece (§6.4) is a half-disc for a round join and degenerate for a bevel, and the miter length is unbounded, so a miter always falls back to the bevel.

A cusp does require the turn angle to come from atan2 (§6.4). The cross product of the tangents vanishes there, so its sign alone would leave the outer side of the corner undetermined.

### 6.9 Zero-Length Subpaths

A zero-length subpath is a single point with no segments (moveto followed by closepath, or lineto to the same point).

Round caps draw a filled circle with radius half the line width. Being built from a point rather than a tangent, the circle has no inherited orientation; it must be wound clockwise like every other piece (§6.1), or it will cancel whatever it overlaps.

Butt and square caps draw nothing, since the tangent is undefined.

A zero-length on-segment in a dash pattern differs: it inherits a tangent from the underlying path. Round caps produce a circle, square caps a square oriented by the tangent, and butt caps produce nothing.

### 6.10 Stroke Self-Intersection

Thick lines along tight curves, self-crossing paths, and dashes landing on top of one another all make a stroke cover the same area more than once.

Rasterise the pieces as they are. Stroke outlines always use nonzero winding, regardless of the fill_rule setting. Since every piece winds the same way (§6.1), overlapping regions receive a winding number of magnitude 2 or higher, clamping to full coverage. The entire stroke interior fills uniformly.

Computing the union of self-intersecting regions adds complexity without improving output.

### 6.11 Assembling the Stroke Outline

This section describes how segments group into pieces.

Emitting one quadrilateral per segment plus one piece per join is always correct, but a flattened curve then costs about seven edges per segment. Most of its joints turn only gently, and there the join hardly matters: runs of such segments merge into strips.

A strip covers consecutive segments with a single polygon: walk forward along the left-hand offset points (for each segment its two endpoints offset by d×N), then backward along the right-hand offset points, and close. Where two segments meet inside a strip, the two offset points on each side are connected by a short chord, and the chord stands in for the join piece.

**Substitution error.** At a joint with turn angle θ the chord differs from the exact join by a second-order sliver: nothing for a bevel (the chord is the bevel), a circular segment of sagitta d × (1 − cos(θ/2)) for a round join, and the omitted miter tip of height d × tan(θ/2) × sin(θ/2) for a miter. For |θ| ≤ 90° all three are below d × sin²θ. The inner-side chord similarly cuts a second-order sliver off the overlap region. A joint may fold only if

    σ × d × sin²θ ≤ ε,

where σ bounds the CTM's largest singular value (the implementation uses the Frobenius norm), so the substitution error in device space stays below the flatness tolerance already accepted by curve flattening (§5).

**Winding safety.** The union-of-pieces argument (§6.1) needs each piece to be simple and clockwise, and a strip could violate this only if one of its offset boundaries doubles back. Two further conditions prevent that. First, per joint,

    d × tan(|θ|/2) ≤ 0.49 × min(L1, L2),

where L1 and L2 are the adjacent segment lengths. The inner offset corner then stays within either half of its segment — equivalently, the offset distance stays below the local radius of curvature — so the cutbacks from a segment's two ends cannot meet, and each offset boundary advances monotonically along its segment. Second, the accumulated turn within one strip is capped at 90°. All tangents of a strip then lie in a quarter-turn cone; both offset boundaries advance strictly in the direction of the cone's axis and stay on opposite sides of the spine, and a polygon made of two disjoint monotone chains joined by its two end edges is simple. It runs forward on the left and back on the right, hence clockwise, like a single segment quadrilateral (§6.2).

A joint that fails any of these tests breaks the strip: emit the strip so far, then the joint's join piece (§6.4), then start a new strip. A polyline meeting at real corners degenerates to one quadrilateral per segment.

For an open subpath the first strip starts at the first segment, and caps (§6.3) terminate both ends. A closed subpath has a joint before every segment, including the seam between the last segment and the first, and strips may cross the seam. Assembly starts at a joint that must break anyway: the first unfoldable joint, or, if the whole loop is gentle, an arbitrary joint — the turn cap, which any closed loop exceeds, forces breaks regardless.

The outline is in user space. The pipeline (§2.4) transforms it to device space before rasterisation.

---

## 7. Summary of Parameters

| Parameter | Notes |
|-----------|-------|
| Flatness tolerance | Device pixels (typical: 0.25–1.0) |
| Miter limit | Dimensionless; at least 1.0 |
| Line cap | Butt, round, or square |
| Line join | Miter, round, or bevel |
| Line width | User-space units; greater than 0 |
| Dash pattern | User-space units; empty array means solid |
| Dash phase | User-space units; offset into pattern |
| Fill rule | Nonzero winding or even-odd |

---

## 8. References

- Sean Barrett, "How the stb_truetype Anti-Aliased Software Rasterizer v2 Works"—signed-area coverage accumulation
- FreeType ftgrays.c—production implementation with extensive comments
- Raph Levien, "Flattening quadratic Béziers"—optimal segment count for quadratics
- Raph Levien, kurbo library (flatten.rs)—Wang's formula for cubics
- Wang, Xiaolin, "Parabolic approximation and best-fit of Bézier curves"—segment count bounds
- Cairo cairo-path-stroke.c—stroke expansion
- PDF Reference Manual—line styles, fill rules, flatness
