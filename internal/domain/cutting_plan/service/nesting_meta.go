package service

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

// Enterprise true-shape (irregular) nesting is driven by a METAHEURISTIC over the
// placement order — the lever DeepNest/SigmaNEST-class nesters rely on. A single
// greedy pass (largest-area first) is order-sensitive and routinely leaves yield on
// the table; searching the order space and keeping the best layout is what lifts
// utilisation on interlocking parts.
//
// nestMetaheuristic evaluates each candidate order with the robust raster
// bottom-left placement (occupancy grid with free rotation, concavity-aware), which
// guarantees non-overlapping, in-bounds layouts for ANY polygon — including concave
// outlines and holes — so the search can never produce an invalid plan. The greedy
// largest-area order seeds the search and is always a candidate, so the result is
// never worse than the one-shot raster nester. The search itself is Iterated Local
// Search (annealing + kicks); a fixed RNG seed keeps plans reproducible/auditable
// (same demand+stock ⇒ same layout).
const (
	metaSeed         int64 = 0x2545F4914F6CDD1D // fixed seed: deterministic, reproducible plans
	metaT0                 = 1.0                // initial annealing temperature (cost units below)
	metaCooling            = 0.997
	metaStagnation         = 1200 // iterations with no improvement before a kick (local optimum)
	metaMaxKicks           = 6    // give up after this many consecutive kicks with no global gain
	metaKickStrength       = 4    // random swaps applied per ILS kick
	metaTimeBudget         = 2 * time.Second
)

func nestMetaheuristic(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	res := pickRes(stock, p.Trim)
	units := expandRasterUnits(demand)
	if len(units) == 0 {
		return buildRasterSolution(nil, nil), nil
	}
	pool := buildRasterPool(stock)

	// Masks depend only on the unit (not the order), so build them once.
	masks := make([][]mask, len(units))
	for i := range units {
		masks[i] = buildMasks(units[i], res)
	}

	decode := func(perm []int) *Solution {
		ou := make([]rasterUnit, len(perm))
		om := make([][]mask, len(perm))
		for i, idx := range perm {
			ou[i], om[i] = units[idx], masks[idx]
		}
		open, unplaced := placeRasterOrder(ou, om, pool, res, p.Trim)
		return buildRasterSolution(open, unplaced)
	}

	// Seed with the greedy largest-area order (the baseline nestRaster uses).
	base := make([]int, len(units))
	for i := range base {
		base[i] = i
	}
	sort.SliceStable(base, func(a, b int) bool { return units[base[a]].areaApx > units[base[b]].areaApx })

	bestSol := decode(base)
	bestCost := solutionCost(bestSol)

	// Nothing to search when a single piece (or a single order) is all there is.
	if len(units) < 3 {
		return bestSol, nil
	}

	// Iterated Local Search: run simulated annealing; when it stagnates, KICK from a
	// perturbation of the best order so far and reheat, diversifying the search across
	// the whole time budget. The global best is always kept, so ILS can only improve on
	// a single annealing run. Deterministic (fixed seed) ⇒ reproducible.
	rng := rand.New(rand.NewSource(metaSeed))
	bestOrder := append([]int(nil), base...)
	cur := append([]int(nil), base...)
	curCost := bestCost
	T := metaT0
	start := time.Now()
	sinceImproved := 0
	kicksWithoutGain := 0
	bestAtLastKick := bestCost
	for time.Since(start) < metaTimeBudget && kicksWithoutGain < metaMaxKicks {
		cand := mutateOrder(cur, rng)
		sol := decode(cand)
		cost := solutionCost(sol)
		sinceImproved++
		if cost < curCost || rng.Float64() < math.Exp((curCost-cost)/T) {
			cur, curCost = cand, cost
			if cost < bestCost {
				bestSol, bestCost, bestOrder = sol, cost, append([]int(nil), cand...)
				sinceImproved = 0
			}
		}
		if T *= metaCooling; T < 1e-4 {
			T = 1e-4
		}
		if sinceImproved >= metaStagnation { // local optimum → kick & reheat from the best
			if bestCost < bestAtLastKick-1e-9 {
				kicksWithoutGain = 0
			} else {
				kicksWithoutGain++
			}
			bestAtLastKick = bestCost
			cur = kickOrder(bestOrder, rng)
			curCost = solutionCost(decode(cur))
			T = metaT0 * 0.5
			sinceImproved = 0
		}
	}
	return bestSol, nil
}

// kickOrder returns a strongly perturbed copy of an order (several random swaps) to
// escape a local optimum while staying near the incumbent — the ILS diversification.
func kickOrder(order []int, rng *rand.Rand) []int {
	out := append([]int(nil), order...)
	n := len(out)
	for k := 0; k < metaKickStrength; k++ {
		i, j := rng.Intn(n), rng.Intn(n)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// mutateOrder returns a copy of perm with a small random move applied: a swap of two
// positions, or a relocation of one piece to another slot (the two neighbourhoods
// that drive packing search).
func mutateOrder(perm []int, rng *rand.Rand) []int {
	n := len(perm)
	out := append([]int(nil), perm...)
	if n < 3 || rng.Intn(2) == 0 {
		i, j := rng.Intn(n), rng.Intn(n)
		out[i], out[j] = out[j], out[i]
		return out
	}
	from := rng.Intn(n)
	v := out[from]
	out = append(out[:from], out[from+1:]...)
	to := rng.Intn(len(out) + 1)
	out = append(out[:to], append([]int{v}, out[to:]...)...)
	return out
}

// solutionCost is the scalar the annealing minimises, normalised to "sheet units" so
// the temperature is meaningful: an unplaced piece costs a whole sheet block, each
// consumed sheet costs 1, and the leftover (1 − utilisation, in [0,1)) is the
// sub-sheet tie-break. A one-sheet swing is therefore ≈ 1.0, matching metaT0, so the
// search genuinely accepts exploratory worse moves early instead of pure hill-climbing.
func solutionCost(s *Solution) float64 {
	return float64(unplacedQty(s))*1e6 + float64(s.StockUsed) + (1 - s.Utilization)
}
