package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service/lp"
)

// optimizer2DCG is the enterprise-grade GUILLOTINE_2D engine. It lifts panel nesting
// (chapas de aço, MDF) from a one-shot free-rectangle heuristic to Gilmore-Gomory
// COLUMN GENERATION, exactly as it does for the 1D bars:
//
//   - the restricted master LP decides how many of each guillotine layout to run,
//     minimising consumed sheet area (remnants discounted so they drain first) while
//     honouring each sheet's availability;
//   - the pricing subproblem is a 2D GUILLOTINE KNAPSACK (guillotineKnapsack) driven
//     by the master's dual prices, producing the most valuable new layout — the cut
//     positions and waste included — which is added and the master re-solved until no
//     layout can improve the relaxation;
//   - the fractional solution is rounded down to whole sheets and the residual is
//     finished with the free-rectangle heuristic core (nest2DGuillotine).
//
// Same safety net as the 1D engine: the heuristic is always computed and the BETTER
// solution returned, so column generation can only improve the result, and any
// internal failure, an over-large discretisation, or a budget timeout falls back to
// the heuristic transparently.
type optimizer2DCG struct{}

func init() { register(optimizer2DCG{}) }

func (optimizer2DCG) Type() entity.CutType { return entity.CutTypeGuillotine2D }

func (optimizer2DCG) Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	heur, err := optimizer2DGuillotine{}.Optimize(demand, stock, p)
	if err != nil {
		return nil, err
	}
	if cg := columnGeneration2D(demand, stock, p); cg != nil && betterSolution(cg, heur) {
		return cg, nil
	}
	return heur, nil
}

// ─── model types ────────────────────────────────────────────────────────────

type cg2DItem struct {
	w, h     float64
	grain    Grain
	allowRot bool
	bound    int
	queue    []cgUnit
}

type cg2DStock struct {
	stockID          int64
	rawW, rawH       float64
	usableW, usableH float64
	cost             float64
	remaining        int
	isRemnant        bool
	pw, ph           int // scaled, inflated plate (usable + kerf)
}

func columnGeneration2D(demand []DemandPiece, stock []StockPiece, p CutParams) (sol *Solution) {
	defer func() {
		if recover() != nil {
			sol = nil
		}
	}()
	if p.Kerf < 0 || p.Trim < 0 || p.MinRemnant < 0 {
		return nil
	}

	// Aggregate demand by (w, h, grain, rotation), keeping the concrete units.
	var items []*cg2DItem
	idx := map[string]int{}
	for _, d := range demand {
		if d.Width <= 0 || d.Height <= 0 || d.Qty <= 0 {
			continue
		}
		grain := d.Grain
		if grain == "" {
			grain = GrainNone
		}
		key := fmt.Sprintf("%.3f|%.3f|%s|%t", d.Width, d.Height, grain, d.AllowRotation)
		i, ok := idx[key]
		if !ok {
			i = len(items)
			idx[key] = i
			items = append(items, &cg2DItem{w: d.Width, h: d.Height, grain: grain, allowRot: d.AllowRotation})
		}
		for n := 0; n < d.Qty; n++ {
			items[i].queue = append(items[i].queue, cgUnit{d.PartID, d.Label})
		}
	}
	for _, it := range items {
		it.bound = len(it.queue)
	}
	if len(items) == 0 {
		return nil
	}

	var stocks []*cg2DStock
	var totalArea float64
	for _, s := range stock {
		if s.Width <= 0 || s.Height <= 0 || s.Qty <= 0 {
			continue
		}
		uw, uh := s.Width-p.Trim, s.Height-p.Trim
		if uw <= 0 || uh <= 0 {
			continue
		}
		cost := s.Width * s.Height
		if s.IsRemnant {
			cost *= remnantCostFactor
		}
		cost *= 1 + float64(s.Priority)*1e-6
		stocks = append(stocks, &cg2DStock{
			stockID: s.StockID, rawW: s.Width, rawH: s.Height,
			usableW: uw, usableH: uh, cost: cost, remaining: s.Qty, isRemnant: s.IsRemnant,
		})
		totalArea += s.Width * s.Height * float64(s.Qty)
	}
	if len(stocks) == 0 {
		return nil
	}

	scale := chooseScale2D(items, stocks, p)
	for _, st := range stocks {
		st.pw = int(math.Floor((st.usableW + p.Kerf) * scale))
		st.ph = int(math.Floor((st.usableH + p.Kerf) * scale))
	}

	// Drop parts that fit no sheet in any allowed orientation.
	var unfit []DemandPiece
	kept := items[:0]
	for _, it := range items {
		if fitsAnySheet(it, stocks) {
			kept = append(kept, it)
			continue
		}
		for _, u := range it.queue {
			unfit = append(unfit, DemandPiece{PartID: u.partID, Label: u.label, Width: it.w, Height: it.h, Qty: 1})
		}
	}
	items = kept
	if len(items) == 0 {
		return nil
	}
	m := len(items)
	M := totalArea + 1

	bounds := make([]int, m)
	srem := make([]int, len(stocks))
	for i, it := range items {
		bounds[i] = it.bound
	}
	for k, st := range stocks {
		srem[k] = st.remaining
	}

	start := time.Now()
	columns := seed2DColumns(items, stocks, scale, p)
	for {
		res := solveMaster(columns, bounds, srem, M)
		if res.Status != lp.Optimal {
			return nil
		}
		pi := res.Dual[:m]
		sigma := res.Dual[m:]

		best := cgColumn{stock: -1}
		bestRC := -cgPricingTol
		for k, st := range stocks {
			// Guard inside the per-sheet loop too: with several heterogeneous sheet
			// sizes a single iteration runs the (possibly expensive) exact DP once per
			// size, which could otherwise overrun the budget before the outer check.
			if time.Since(start) > cgTimeBudget {
				break
			}
			counts, z, layout, waste := priceSheet(items, pi, st, bounds, scale, p)
			if rc := st.cost - z - sigma[k]; rc < bestRC {
				bestRC = rc
				best = cgColumn{stock: k, comp: append([]int(nil), counts...), cost: st.cost, layout: layout, waste: waste}
			}
		}
		if best.stock == -1 || columnExists(columns, best) {
			break
		}
		columns = append(columns, best)
		if len(columns) >= cgMaxColumns || time.Since(start) > cgTimeBudget {
			break
		}
	}

	final := solveMaster(columns, bounds, srem, M)
	if final.Status != lp.Optimal {
		return nil
	}
	return recoverInteger2D(columns, final.X, items, stocks, p, unfit)
}

// priceSheet generates the most valuable layout for one sheet from the current dual
// prices. It uses the EXACT guillotine knapsack when the sheet's discretisation fits
// the DP budget (optimal patterns on small/medium sheets), and otherwise a fast
// dual-weighted heuristic packer, so column generation always produces a column and
// scales to full-size panels (where the exact DP would be intractable).
func priceSheet(items []*cg2DItem, pi []float64, st *cg2DStock, bounds []int, scale float64, p CutParams) (counts []int, value float64, layout []cg2DPlace, waste []freeRect) {
	opts := buildOptions(items, pi, st, scale, p.Kerf)
	if c, z, places, w, ok := guillotineKnapsack(opts, st.pw, st.ph, len(items)); ok {
		return c, z, toRealPlaces(places, scale, p.Trim), toRealWaste(w, scale, p.Trim)
	}
	return packSheetByValue(items, pi, st, bounds, p)
}

// packSheetByValue is the heuristic pricer: it fills one sheet greedily, placing the
// highest-dual parts first into the best-fitting free rectangle (the same guillotine
// free-rectangle machinery the heuristic optimiser uses), bounded by demand. It
// returns the packed counts, their total dual value, the layout and the leftover
// rectangles — all in real (mm) coordinates.
func packSheetByValue(items []*cg2DItem, duals []float64, st *cg2DStock, bounds []int, p CutParams) (counts []int, value float64, layout []cg2DPlace, waste []freeRect) {
	m := len(items)
	counts = make([]int, m)
	s := &sheet{free: []freeRect{{x: p.Trim, y: p.Trim, w: st.usableW, h: st.usableH}}}

	order := make([]int, 0, m)
	for i := range items {
		if duals[i] > 0 {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return duals[order[a]] > duals[order[b]] })

	for _, i := range order {
		it := items[i]
		canRot := it.allowRot && (it.grain == "" || it.grain == GrainNone)
		for n := 0; n < bounds[i]; n++ {
			u := unit2D{partID: int64(i), w: it.w, h: it.h, canRotate: canRot}
			if !placeBestFit([]*sheet{s}, u, p.Kerf) {
				break // no room left for this (or any smaller copy of this) item
			}
			counts[i]++
		}
	}
	for _, pl := range s.placements {
		i := int(pl.PartID)
		value += duals[i]
		layout = append(layout, cg2DPlace{item: i, x: pl.X, y: pl.Y, w: pl.W, h: pl.H, rotated: pl.Rotated})
	}
	waste = append(waste, s.free...)
	return counts, value, layout, waste
}

// buildOptions enumerates the placeable orientations of each item on a sheet, valued
// at the item's dual price, in scaled inflated units (edge + kerf).
func buildOptions(items []*cg2DItem, pi []float64, st *cg2DStock, scale, kerf float64) []gkOption {
	var opts []gkOption
	for i, it := range items {
		v := pi[i]
		if v <= 0 {
			continue // a non-positive dual never improves the pattern
		}
		canRot := it.allowRot && (it.grain == "" || it.grain == GrainNone)
		add := func(w, h float64, rotated bool) {
			ow := int(math.Ceil((w + kerf) * scale))
			oh := int(math.Ceil((h + kerf) * scale))
			if ow <= st.pw && oh <= st.ph {
				opts = append(opts, gkOption{item: i, w: ow, h: oh, rw: w, rh: h, rotated: rotated, value: v})
			}
		}
		add(it.w, it.h, false)
		if canRot && it.w != it.h {
			add(it.h, it.w, true)
		}
	}
	return opts
}

func toRealPlaces(places []gkPlace, scale, trim float64) []cg2DPlace {
	out := make([]cg2DPlace, len(places))
	for i, pl := range places {
		out[i] = cg2DPlace{
			item: pl.item, x: trim + float64(pl.x)/scale, y: trim + float64(pl.y)/scale,
			w: pl.rw, h: pl.rh, rotated: pl.rotated,
		}
	}
	return out
}

func toRealWaste(waste []gkRect, scale, trim float64) []freeRect {
	out := make([]freeRect, 0, len(waste))
	for _, r := range waste {
		out = append(out, freeRect{
			x: trim + float64(r.x)/scale, y: trim + float64(r.y)/scale,
			w: float64(r.w) / scale, h: float64(r.h) / scale,
		})
	}
	return out
}

// recoverInteger2D turns the LP column pool into whole sheets via the cost-effective
// greedy cover (roundCover), finishes any residual demand with the heuristic core,
// and assembles the final solution.
func recoverInteger2D(columns []cgColumn, x []float64, items []*cg2DItem, stocks []*cg2DStock, p CutParams, unfit []DemandPiece) *Solution {
	m := len(items)
	itemArea := make([]float64, m)
	bounds := make([]int, m)
	for i, it := range items {
		itemArea[i] = it.w * it.h
		bounds[i] = it.bound
	}
	srem := make([]int, len(stocks))
	for k, st := range stocks {
		srem[k] = st.remaining
	}

	picks, _, remStock := roundCover(columns, bounds, srem, itemArea)

	cursor := make([]int, m)
	var open []*sheet
	for _, pk := range picks {
		col := columns[pk.col]
		st := stocks[col.stock]
		for cp := 0; cp < pk.copies; cp++ {
			open = append(open, materialize2DSheet(st, col, items, cursor, pk.place))
		}
	}

	var residualDemand []DemandPiece
	for i, it := range items {
		for c := cursor[i]; c < len(it.queue); c++ {
			u := it.queue[c]
			residualDemand = append(residualDemand, DemandPiece{
				PartID: u.partID, Label: u.label, Width: it.w, Height: it.h,
				Grain: it.grain, AllowRotation: it.allowRot, Qty: 1,
			})
		}
	}
	var residualStock []StockPiece
	for k, st := range stocks {
		if remStock[k] > 0 {
			residualStock = append(residualStock, StockPiece{
				StockID: st.stockID, Width: st.rawW, Height: st.rawH,
				Qty: remStock[k], IsRemnant: st.isRemnant,
			})
		}
	}

	resSheets, resUnplaced, err := nest2DGuillotine(residualDemand, residualStock, p)
	if err != nil {
		return nil
	}
	open = append(open, resSheets...)

	unplaced := append([]DemandPiece(nil), unfit...)
	unplaced = append(unplaced, resUnplaced...)
	return buildSolution2D(open, unplaced, p)
}

// materialize2DSheet lays a column's guillotine layout onto one fresh sheet, placing
// only `place[i]` units of each item (a full pattern, or a trimmed subset), drawing
// concrete part ids/labels from each item's queue via the shared cursor.
func materialize2DSheet(st *cg2DStock, col cgColumn, items []*cg2DItem, cursor, place []int) *sheet {
	s := &sheet{stockID: st.stockID, width: st.rawW, height: st.rawH, isRemnant: st.isRemnant}
	placed := make([]int, len(items))
	for _, pl := range col.layout {
		if placed[pl.item] >= place[pl.item] {
			continue // trimmed pattern: this copy of the item is not needed
		}
		placed[pl.item]++
		it := items[pl.item]
		u := it.queue[cursor[pl.item]]
		cursor[pl.item]++
		s.placements = append(s.placements, Placement{
			PartID: u.partID, Label: u.label, X: pl.x, Y: pl.y, W: pl.w, H: pl.h, Rotated: pl.rotated,
		})
		s.usedArea += pl.w * pl.h
	}
	s.free = append(s.free, col.waste...)
	return s
}

// seed2DColumns primes the pool with one single-part pattern per item, on the
// cheapest sheet that holds it. These granular columns make the greedy rounding pick
// the right sheet for small leftovers; column generation then enriches the pool with
// dense multi-part layouts.
func seed2DColumns(items []*cg2DItem, stocks []*cg2DStock, scale float64, p CutParams) []cgColumn {
	var cols []cgColumn
	for i, it := range items {
		canRot := it.allowRot && (it.grain == "" || it.grain == GrainNone)
		bestK := -1
		var bw, bh float64
		var brot bool
		for k, st := range stocks {
			if it.w <= st.usableW+eps && it.h <= st.usableH+eps {
				if bestK == -1 || st.cost < stocks[bestK].cost {
					bestK, bw, bh, brot = k, it.w, it.h, false
				}
			} else if canRot && it.h <= st.usableW+eps && it.w <= st.usableH+eps {
				if bestK == -1 || st.cost < stocks[bestK].cost {
					bestK, bw, bh, brot = k, it.h, it.w, true
				}
			}
		}
		if bestK == -1 {
			continue
		}
		st := stocks[bestK]
		comp := make([]int, len(items))
		comp[i] = 1
		layout := []cg2DPlace{{item: i, x: p.Trim, y: p.Trim, w: bw, h: bh, rotated: brot}}
		var waste []freeRect
		if r := st.usableW - bw; r > eps {
			waste = append(waste, freeRect{x: p.Trim + bw, y: p.Trim, w: r, h: st.usableH})
		}
		if t := st.usableH - bh; t > eps {
			waste = append(waste, freeRect{x: p.Trim, y: p.Trim + bh, w: bw, h: t})
		}
		cols = append(cols, cgColumn{stock: bestK, comp: comp, cost: st.cost, layout: layout, waste: waste})
	}
	return cols
}

// ─── helpers ────────────────────────────────────────────────────────────────

func fitsAnySheet(it *cg2DItem, stocks []*cg2DStock) bool {
	canRot := it.allowRot && (it.grain == "" || it.grain == GrainNone)
	for _, st := range stocks {
		if it.w <= st.usableW+eps && it.h <= st.usableH+eps {
			return true
		}
		if canRot && it.h <= st.usableW+eps && it.w <= st.usableH+eps {
			return true
		}
	}
	return false
}

func chooseScale2D(items []*cg2DItem, stocks []*cg2DStock, p CutParams) float64 {
	whole := func(v float64) bool { return math.Abs(v-math.Round(v)) < 1e-6 }
	integral := whole(p.Kerf) && whole(p.Trim)
	for _, it := range items {
		if !whole(it.w) || !whole(it.h) {
			integral = false
		}
	}
	for _, st := range stocks {
		if !whole(st.rawW) || !whole(st.rawH) {
			integral = false
		}
	}
	if integral {
		return 1
	}
	return 10
}
