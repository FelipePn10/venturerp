package service

import (
	"math"
	"testing"
)

func TestGeom_RotatePolyNormalises(t *testing.T) {
	// A 100×10 rectangle rotated 45° has a ~77.8 square bbox, anchored at origin.
	r := rotatePoly([]Point{{0, 0}, {100, 0}, {100, 10}, {0, 10}}, 45)
	minX, minY, maxX, maxY := polyBounds(r)
	if math.Abs(minX) > 1e-6 || math.Abs(minY) > 1e-6 {
		t.Fatalf("not anchored at origin: min=(%.3f,%.3f)", minX, minY)
	}
	want := (100 + 10) / math.Sqrt2
	if math.Abs(maxX-want) > 1e-3 || math.Abs(maxY-want) > 1e-3 {
		t.Fatalf("bbox = %.3f×%.3f, want %.3f", maxX, maxY, want)
	}
}

func TestGeom_DiagonalFitsViaRotation(t *testing.T) {
	// A 130×10 bar does not fit a 100×100 sheet axis-aligned, but at 45° its bbox is
	// ~99×99 and it does — the property the free-rotation nester exploits.
	bar := []Point{{0, 0}, {130, 0}, {130, 10}, {0, 10}}
	_, _, w0, _ := polyBounds(bar)
	if w0 <= 100 {
		t.Fatal("test premise wrong: bar already fits axis-aligned")
	}
	r := rotatePoly(bar, 45)
	_, _, w, h := polyBounds(r)
	if w > 100 || h > 100 {
		t.Fatalf("rotated bar bbox %.2f×%.2f should fit 100×100", w, h)
	}
}
