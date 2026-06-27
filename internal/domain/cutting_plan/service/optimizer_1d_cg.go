package service

import (
	"math"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service/lp"
)

// optimizer1DCG is the enterprise-grade LINEAR_1D engine. It solves the classic
// one-dimensional cutting-stock problem with Gilmore-Gomory COLUMN GENERATION —
// the method behind SAP/Oracle/Focco-class optimisers:
//
//   - a restricted master LP decides how many of each cutting pattern to run,
//     minimising consumed raw stock (with reusable remnants discounted so they are
//     drained first) while honouring each stock length's availability;
//   - the pricing subproblem — a bounded knapsack fed by the master's dual prices —
//     generates the single most profitable new pattern, which is added and the
//     master re-solved, until no pattern can improve the LP (provable optimality of
//     the relaxation);
//   - the fractional LP solution is rounded down to whole bars and the small
//     integer residual is finished with the Best-Fit Decreasing core (nest1D),
//     yielding a complete, availability-feasible plan typically within one bar of
//     the true optimum.
//
// The engine is wrapped in a safety net: the BFD heuristic is always computed and
// the BETTER of the two solutions is returned, so column generation can only ever
// improve the result — never regress it — and any internal failure or budget
// timeout falls back transparently to BFD.
type optimizer1DCG struct{}

func init() { register(optimizer1DCG{}) }

func (optimizer1DCG) Type() entity.CutType { return entity.CutTypeLinear1D }

const (
	cgTimeBudget      = 5 * time.Second
	cgMaxColumns      = 4000
	cgScaleCap        = 4_000_000 // bound on a scaled bar capacity (knapsack DP size)
	cgPricingTol      = 1e-4      // a pattern must improve the LP by at least this
	remnantCostFactor = 1e-3      // remnants are near-free so the LP drains them first
)

func (optimizer1DCG) Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	// BFD validates the parameters and is the guaranteed baseline.
	bfd, err := optimizer1D{}.Optimize(demand, stock, p)
	if err != nil {
		return nil, err
	}
	if cg := columnGeneration(demand, stock, p); cg != nil && betterSolution(cg, bfd) {
		return cg, nil
	}
	return bfd, nil
}

// ─── model types ────────────────────────────────────────────────────────────

type cgUnit struct {
	partID int64
	label  string
}

type cgItem struct {
	length float64
	weight int      // scaled (length + kerf), rounded UP so a pattern never overfills
	bound  int      // total demanded quantity
	queue  []cgUnit // concrete units, popped as they are placed
}

type cgStock struct {
	stockID   int64
	rawLen    float64
	usable    float64
	capacity  int     // scaled (usable + kerf), rounded DOWN (conservative)
	cost      float64 // LP cost of one bar (raw length, remnants discounted)
	remaining int
	isRemnant bool
}

type cgColumn struct {
	stock  int
	comp   []int
	cost   float64
	layout []cg2DPlace // 2D only: the guillotine layout backing this pattern (nil for 1D)
	waste  []freeRect  // 2D only: the layout's leftover rectangles (for remnant metrics)
}

// cg2DPlace is one part positioned on a sheet, in real (mm) coordinates, produced by
// the 2D guillotine pricer and replayed when a 2D pattern is materialised.
type cg2DPlace struct {
	item    int
	x, y    float64
	w, h    float64
	rotated bool
}

// columnGeneration runs the full pipeline and returns a complete solution, or nil
// to signal "use the BFD baseline" (degenerate input or an unrecoverable state).
func columnGeneration(demand []DemandPiece, stock []StockPiece, p CutParams) (sol *Solution) {
	// Never let an internal edge case crash the optimiser: fall back to BFD.
	defer func() {
		if recover() != nil {
			sol = nil
		}
	}()
	if p.Kerf < 0 || p.Trim < 0 || p.MinRemnant < 0 {
		return nil
	}

	// Aggregate demand by length, keeping the concrete units for placement.
	var items []*cgItem
	idxByKey := map[int64]int{}
	for _, d := range demand {
		if d.Length <= 0 || d.Qty <= 0 {
			continue
		}
		key := int64(math.Round(d.Length * 1000))
		idx, ok := idxByKey[key]
		if !ok {
			idx = len(items)
			idxByKey[key] = idx
			items = append(items, &cgItem{length: d.Length})
		}
		for i := 0; i < d.Qty; i++ {
			items[idx].queue = append(items[idx].queue, cgUnit{d.PartID, d.Label})
		}
	}
	for _, it := range items {
		it.bound = len(it.queue)
	}
	if len(items) == 0 {
		return nil
	}

	var stocks []*cgStock
	var totalRaw float64
	for _, s := range stock {
		if s.Length <= 0 || s.Qty <= 0 {
			continue
		}
		usable := s.Length - p.Trim
		if usable <= 0 {
			continue
		}
		cost := s.Length
		if s.IsRemnant {
			cost *= remnantCostFactor
		}
		cost *= 1 + float64(s.Priority)*1e-6 // gentle tie-break: lower priority first
		stocks = append(stocks, &cgStock{
			stockID: s.StockID, rawLen: s.Length, usable: usable,
			cost: cost, remaining: s.Qty, isRemnant: s.IsRemnant,
		})
		totalRaw += s.Length * float64(s.Qty)
	}
	if len(stocks) == 0 {
		return nil
	}

	scale := chooseScale(items, stocks, p)
	maxCap := 0
	for _, st := range stocks {
		st.capacity = int(math.Floor((st.usable + p.Kerf) * scale))
		if st.capacity > maxCap {
			maxCap = st.capacity
		}
	}

	// Drop pieces that fit in no stock at all; they are reported as unplaced.
	var unfit []DemandPiece
	kept := items[:0]
	for _, it := range items {
		it.weight = int(math.Ceil((it.length + p.Kerf) * scale))
		if it.weight > maxCap {
			for _, u := range it.queue {
				unfit = appendDemand(unfit, u.partID, u.label, it.length)
			}
			continue
		}
		kept = append(kept, it)
	}
	items = kept
	if len(items) == 0 {
		return nil
	}
	m := len(items)

	// Shortfall penalty: covering a piece (≤ one bar) must always beat leaving it
	// short, so M dominates any stock cost.
	M := totalRaw + 1

	bounds := make([]int, m)
	srem := make([]int, len(stocks))
	for i, it := range items {
		bounds[i] = it.bound
	}
	for k, st := range stocks {
		srem[k] = st.remaining
	}

	// ─── column-generation loop ───────────────────────────────────────────────
	start := time.Now()
	var columns []cgColumn
	for {
		res := solveMaster(columns, bounds, srem, M)
		if res.Status != lp.Optimal {
			return nil
		}
		pi := res.Dual[:m]    // demand duals (≥ 0)
		sigma := res.Dual[m:] // stock-availability duals (≤ 0)

		best := cgColumn{stock: -1}
		bestRC := -cgPricingTol
		vals := make([]float64, m)
		ws := make([]int, m)
		bs := make([]int, m)
		for i, it := range items {
			ws[i], bs[i] = it.weight, it.bound
		}
		for k, st := range stocks {
			for i := range items {
				v := pi[i]
				if v < 0 {
					v = 0
				}
				vals[i] = v
			}
			counts, z := boundedKnapsack(vals, ws, bs, st.capacity)
			if rc := st.cost - z - sigma[k]; rc < bestRC {
				bestRC = rc
				best = cgColumn{stock: k, comp: append([]int(nil), counts...), cost: st.cost}
			}
		}
		if best.stock == -1 || columnExists(columns, best) {
			break // LP optimal (or no new pattern) — stop generating
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

	return recoverInteger(columns, final.X, items, stocks, p, unfit)
}

// solveMaster assembles and solves the restricted master LP for the current set of
// pattern columns. It is dimension-agnostic (shared by the 1D and 2D engines):
// variables are the pattern run-counts followed by one shortfall variable per item;
// constraints are demand (≥, RHS = bounds) then stock availability (≤, RHS =
// stockRemaining).
func solveMaster(columns []cgColumn, bounds []int, stockRemaining []int, M float64) lp.Result {
	m := len(bounds)
	k := len(stockRemaining)
	nPat := len(columns)
	n := nPat + m
	rows := m + k

	A := make([][]float64, rows)
	for r := 0; r < rows; r++ {
		A[r] = make([]float64, n)
	}
	C := make([]float64, n)
	B := make([]float64, rows)
	typ := make([]lp.ConstraintType, rows)

	for j, col := range columns {
		for i, c := range col.comp {
			if c != 0 {
				A[i][j] = float64(c)
			}
		}
		A[m+col.stock][j] = 1
		C[j] = col.cost
	}
	for i := 0; i < m; i++ {
		A[i][nPat+i] = 1 // shortfall variable
		C[nPat+i] = M
		B[i] = float64(bounds[i])
		typ[i] = lp.GE
	}
	for kk := 0; kk < k; kk++ {
		B[m+kk] = float64(stockRemaining[kk])
		typ[m+kk] = lp.LE
	}
	return lp.Solve(lp.Problem{C: C, A: A, B: B, Type: typ})
}

// recoverInteger rounds the LP solution down to whole bars (respecting stock
// availability and avoiding over-production), finishes the residual demand with the
// BFD core, and assembles the final solution.
func recoverInteger(columns []cgColumn, x []float64, items []*cgItem, stocks []*cgStock, p CutParams, unfit []DemandPiece) *Solution {
	m := len(items)
	remStock := make([]int, len(stocks))
	for k := range stocks {
		remStock[k] = stocks[k].remaining
	}
	remDemand := make([]int, m)
	for i := range items {
		remDemand[i] = items[i].bound
	}

	// Apply patterns largest-LP-value first.
	order := make([]int, len(columns))
	for j := range order {
		order[j] = j
	}
	sort.SliceStable(order, func(a, b int) bool { return x[order[a]] > x[order[b]] })

	cursor := make([]int, m) // next unplaced concrete unit per item
	var bins []*bin
	for _, j := range order {
		col := columns[j]
		t := int(math.Floor(x[j] + 1e-9))
		if t <= 0 {
			continue
		}
		if t > remStock[col.stock] {
			t = remStock[col.stock]
		}
		for i, c := range col.comp { // cap to avoid over-producing any item
			if c > 0 {
				if lim := remDemand[i] / c; lim < t {
					t = lim
				}
			}
		}
		if t <= 0 {
			continue
		}
		st := stocks[col.stock]
		for cp := 0; cp < t; cp++ {
			bins = append(bins, materializeBin(st, col.comp, items, cursor, p))
		}
		remStock[col.stock] -= t
		for i, c := range col.comp {
			remDemand[i] -= t * c
		}
	}

	// Residual demand = the units no pattern consumed; finish them with BFD on the
	// stock still available.
	var residualDemand []DemandPiece
	for i, it := range items {
		for c := cursor[i]; c < len(it.queue); c++ {
			u := it.queue[c]
			residualDemand = appendDemand(residualDemand, u.partID, u.label, it.length)
		}
	}
	var residualStock []StockPiece
	for k, st := range stocks {
		if remStock[k] > 0 {
			residualStock = append(residualStock, StockPiece{
				StockID: st.stockID, Length: st.rawLen, Qty: remStock[k],
				IsRemnant: st.isRemnant,
			})
		}
	}

	resBins, resUnplaced, err := nest1D(residualDemand, residualStock, p)
	if err != nil {
		return nil
	}
	bins = append(bins, resBins...)

	unplaced := unfit
	for _, d := range resUnplaced {
		// resUnplaced entries are already aggregated (d.Qty may be > 1); add the whole
		// quantity, not one piece per entry.
		for i := 0; i < d.Qty; i++ {
			unplaced = appendDemand(unplaced, d.PartID, d.Label, d.Length)
		}
	}
	return buildSolution(bins, unplaced, p)
}

// materializeBin lays a pattern's pieces onto one fresh bar, longest first, drawing
// concrete part ids/labels from each item's queue via the shared cursor.
func materializeBin(st *cgStock, comp []int, items []*cgItem, cursor []int, p CutParams) *bin {
	b := &bin{stockID: st.stockID, length: st.rawLen, usable: st.usable, isRemnant: st.isRemnant}
	type piece struct {
		item   int
		length float64
	}
	var pieces []piece
	for i, c := range comp {
		for n := 0; n < c; n++ {
			pieces = append(pieces, piece{item: i, length: items[i].length})
		}
	}
	sort.SliceStable(pieces, func(a, b int) bool { return pieces[a].length > pieces[b].length })
	for _, pc := range pieces {
		it := items[pc.item]
		u := it.queue[cursor[pc.item]]
		cursor[pc.item]++
		placeInBin(b, u.partID, u.label, it.length, p.Trim, p.Kerf)
	}
	return b
}

// ─── helpers ────────────────────────────────────────────────────────────────

func columnExists(columns []cgColumn, c cgColumn) bool {
	for _, x := range columns {
		if x.stock != c.stock || len(x.comp) != len(c.comp) {
			continue
		}
		same := true
		for i := range x.comp {
			if x.comp[i] != c.comp[i] {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}
	return false
}

// chooseScale picks the integer scaling factor for the knapsack: 1 when every
// length/kerf/trim is already a whole millimetre (the common case), else a finer
// sub-millimetre grid, clamped so the DP capacity stays bounded.
func chooseScale(items []*cgItem, stocks []*cgStock, p CutParams) float64 {
	whole := func(v float64) bool { return math.Abs(v-math.Round(v)) < 1e-6 }
	integral := whole(p.Kerf) && whole(p.Trim)
	for _, it := range items {
		if !whole(it.length) {
			integral = false
		}
	}
	for _, st := range stocks {
		if !whole(st.rawLen) {
			integral = false
		}
	}
	scale := 1.0
	if !integral {
		scale = 10.0
	}
	maxUsable := 0.0
	for _, st := range stocks {
		if u := st.usable + p.Kerf; u > maxUsable {
			maxUsable = u
		}
	}
	for scale > 1 && maxUsable*scale > cgScaleCap {
		scale /= 10
	}
	if maxUsable*scale > cgScaleCap && maxUsable > 0 {
		scale = cgScaleCap / maxUsable
	}
	if scale <= 0 {
		scale = 1
	}
	return scale
}

// betterSolution reports whether a is strictly better than b: fewer unplaced pieces
// first, then less raw stock consumed (higher utilisation), then fewer bars.
func betterSolution(a, b *Solution) bool {
	ua, ub := unplacedQty(a), unplacedQty(b)
	if ua != ub {
		return ua < ub
	}
	if math.Abs(a.TotalStock-b.TotalStock) > 1e-6 {
		return a.TotalStock < b.TotalStock
	}
	return a.StockUsed < b.StockUsed
}

func unplacedQty(s *Solution) int {
	n := 0
	for _, d := range s.Unplaced {
		n += d.Qty
	}
	return n
}
