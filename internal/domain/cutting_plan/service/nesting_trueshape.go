package service

import (
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// trueShapeBBox is the native true-shape (irregular) nester. True polygon nesting
// (no-fit-polygon) is expensive to do well; this provider gives a usable baseline
// out of the box by enclosing each part in its axis-aligned bounding box and
// reusing the 2D guillotine optimiser. It is registered as the default for
// CutTypeTrueShape2D, so true-shape plans always produce a result.
//
// For full true-shape yield (interlocking concavities, laser/plasma), an external
// nesting engine (e.g. DeepNest / ProNest) is plugged in via the same
// CuttingOptimizer contract and overrides this default — see the infrastructure
// HTTP nesting adapter.
type trueShapeBBox struct{}

func (trueShapeBBox) Type() entity.CutType { return entity.CutTypeTrueShape2D }

func (trueShapeBBox) Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	// Enclose each part in its bounding box when only a polygon is given.
	bb := make([]DemandPiece, len(demand))
	for i, d := range demand {
		d2 := d
		if (d2.Width <= 0 || d2.Height <= 0) && len(d2.Polygon) >= 3 {
			d2.Width, d2.Height = PolygonBBox(d2.Polygon)
		}
		bb[i] = d2
	}

	g, err := Optimizer(entity.CutTypeGuillotine2D)
	if err != nil {
		return nil, errors.New("true-shape bbox provider requires the 2D guillotine optimiser")
	}
	sol, err := g.Optimize(bb, stock, p)
	if err != nil {
		return nil, err
	}
	// Carry the 90° rotation through as an explicit angle.
	for pi := range sol.Patterns {
		for j := range sol.Patterns[pi].Placements {
			if sol.Patterns[pi].Placements[j].Rotated {
				sol.Patterns[pi].Placements[j].RotationDeg = 90
			}
		}
	}
	return sol, nil
}

// PolygonBBox returns the width and height of a polygon's axis-aligned bounding box.
func PolygonBBox(poly []Point) (w, h float64) {
	if len(poly) == 0 {
		return 0, 0
	}
	minX, minY := poly[0].X, poly[0].Y
	maxX, maxY := poly[0].X, poly[0].Y
	for _, pt := range poly[1:] {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}
	return maxX - minX, maxY - minY
}
