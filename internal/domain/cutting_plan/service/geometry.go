package service

import "math"

// Geometry helpers for free-rotation true-shape nesting. The nester rasterises pieces
// at arbitrary angles onto an occupancy grid (sound collision — it can never miss an
// overlap), so the only continuous geometry it needs is rotating a polygon and reading
// its bounding box; the heavy/fragile polygon-overlap arithmetic is deliberately
// avoided in favour of the grid.

// sinCosDeg returns sin and cos of angleDeg, using EXACT values for multiples of 90°
// so cardinal rotations introduce no floating-point residual (which would otherwise
// inflate a rasterised bounding box by a cell and break cell-perfect interlocking).
func sinCosDeg(angleDeg float64) (sin, cos float64) {
	switch math.Mod(angleDeg, 360) {
	case 0:
		return 0, 1
	case 90:
		return 1, 0
	case 180:
		return 0, -1
	case 270:
		return -1, 0
	}
	rad := angleDeg * math.Pi / 180
	return math.Sin(rad), math.Cos(rad)
}

// rotatePoly rotates a polygon by angleDeg about the origin and shifts it so its
// bounding box starts at (0,0) — the canonical form used for rasterisation.
func rotatePoly(poly []Point, angleDeg float64) []Point {
	s, c := sinCosDeg(angleDeg)
	out := make([]Point, len(poly))
	for i, p := range poly {
		out[i] = Point{X: p.X*c - p.Y*s, Y: p.X*s + p.Y*c}
	}
	minX, minY := out[0].X, out[0].Y
	for _, p := range out[1:] {
		minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
	}
	for i := range out {
		out[i].X -= minX
		out[i].Y -= minY
	}
	return out
}

// polyBounds returns the axis-aligned bounding box of poly.
func polyBounds(poly []Point) (minX, minY, maxX, maxY float64) {
	minX, minY = poly[0].X, poly[0].Y
	maxX, maxY = poly[0].X, poly[0].Y
	for _, p := range poly[1:] {
		minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
		maxX, maxY = math.Max(maxX, p.X), math.Max(maxY, p.Y)
	}
	return
}
