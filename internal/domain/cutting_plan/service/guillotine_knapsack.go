package service

import "sort"

// guillotineKnapsack is the pricing subproblem of the 2D column generation: given
// the dual prices of the demand constraints it finds the most valuable set of
// rectangles that can be GUILLOTINE-cut from one sheet — the two-dimensional
// analogue of the 1D knapsack pricer.
//
// It is the classic Gilmore-Gomory guillotine dynamic program over a discretised
// plate. The plate's reachable edge lengths are restricted to the "normal patterns"
// (subset sums of the part edge lengths), and
//
//	V(p,q) = max( best single part fitting p×q,
//	              max over vertical   cut x : V(x,q) + V(p−x,q),
//	              max over horizontal cut y : V(p,y) + V(p,q−y) )
//
// Each part is valued at its dual price, so the DP maximises Σ dualᵢ·countᵢ. The
// chosen layout is rebuilt from the per-state decisions (positions and the waste
// rectangles left over), which the caller replays as real cuts.
//
// Lengths are integers scaled by the caller; kerf is folded in by inflation (parts
// and the plate are each enlarged by one kerf), so cuts stay kerf-correct. The DP
// is bounded by hard caps on the discretisation; if an instance exceeds them the
// function reports ok=false and the caller falls back to the heuristic.

const (
	// The DP cost is ≈ states·(|X|+|Y|)/2, so the STATE cap is the real budget; the
	// axis cap only guards the per-state inner loop (one very long axis forces the
	// other short, since their product is capped).
	gk2DAxisCap  = 2000    // max discretisation points per axis
	gk2DStateCap = 120_000 // max DP states (|X|·|Y|)
)

// gkOption is one placeable orientation of a part on a given sheet, in scaled
// (inflated) units, carrying the part's real dimensions for placement.
type gkOption struct {
	item    int
	w, h    int // scaled, inflated (edge + kerf)
	rw, rh  float64
	rotated bool
	value   float64 // dual price of the part
}

// gkPlace is one placed part in scaled (inflated) plate coordinates.
type gkPlace struct {
	item    int
	x, y    int
	rw, rh  float64
	rotated bool
}

// gkRect is a leftover (waste) rectangle in scaled (inflated) coordinates.
type gkRect struct{ x, y, w, h int }

type gkKind uint8

const (
	gkEmpty gkKind = iota
	gkLeaf
	gkVCut
	gkHCut
)

type gkDecision struct {
	kind     gkKind
	opt      int // option index (leaf)
	split    int // child index in Xset/Yset (cut)
	rightIdx int // floor index of the right/top child
}

func guillotineKnapsack(opts []gkOption, PW, PH, nItems int) (counts []int, value float64, places []gkPlace, waste []gkRect, ok bool) {
	if PW <= 0 || PH <= 0 || len(opts) == 0 {
		return make([]int, nItems), 0, nil, nil, true
	}

	wDims := distinctDims(opts, true)
	hDims := distinctDims(opts, false)
	Xset := subsetSums(wDims, PW)
	Yset := subsetSums(hDims, PH)
	if len(Xset) > gk2DAxisCap || len(Yset) > gk2DAxisCap || len(Xset)*len(Yset) > gk2DStateCap {
		return nil, 0, nil, nil, false
	}
	nx, ny := len(Xset), len(Yset)

	floorX := func(v int) int { return floorIndex(Xset, v) }
	floorY := func(v int) int { return floorIndex(Yset, v) }

	V := make([][]float64, nx)
	dec := make([][]gkDecision, nx)
	for i := range V {
		V[i] = make([]float64, ny)
		dec[i] = make([]gkDecision, ny)
	}

	for i := 0; i < nx; i++ {
		p := Xset[i]
		for j := 0; j < ny; j++ {
			q := Yset[j]
			best := 0.0
			bd := gkDecision{kind: gkEmpty}

			for oi := range opts {
				if opts[oi].w <= p && opts[oi].h <= q && opts[oi].value > best {
					best = opts[oi].value
					bd = gkDecision{kind: gkLeaf, opt: oi}
				}
			}
			for k := 1; k < nx && Xset[k]*2 <= p; k++ {
				ri := floorX(p - Xset[k])
				if cand := V[k][j] + V[ri][j]; cand > best {
					best = cand
					bd = gkDecision{kind: gkVCut, split: k, rightIdx: ri}
				}
			}
			for k := 1; k < ny && Yset[k]*2 <= q; k++ {
				ri := floorY(q - Yset[k])
				if cand := V[i][k] + V[i][ri]; cand > best {
					best = cand
					bd = gkDecision{kind: gkHCut, split: k, rightIdx: ri}
				}
			}
			V[i][j] = best
			dec[i][j] = bd
		}
	}

	ti, tj := floorX(PW), floorY(PH)
	counts = make([]int, nItems)

	var expand func(i, j, offX, offY int)
	expand = func(i, j, offX, offY int) {
		d := dec[i][j]
		switch d.kind {
		case gkEmpty:
			if Xset[i] > 0 && Yset[j] > 0 {
				waste = append(waste, gkRect{offX, offY, Xset[i], Yset[j]})
			}
		case gkLeaf:
			o := opts[d.opt]
			counts[o.item]++
			places = append(places, gkPlace{item: o.item, x: offX, y: offY, rw: o.rw, rh: o.rh, rotated: o.rotated})
			// Leftover within this leaf: the strip right of the part and the strip
			// above it (guillotine-consistent).
			if r := Xset[i] - o.w; r > 0 {
				waste = append(waste, gkRect{offX + o.w, offY, r, Yset[j]})
			}
			if t := Yset[j] - o.h; t > 0 {
				waste = append(waste, gkRect{offX, offY + o.h, o.w, t})
			}
		case gkVCut:
			lw := Xset[d.split]
			expand(d.split, j, offX, offY)
			expand(d.rightIdx, j, offX+lw, offY)
		case gkHCut:
			lh := Yset[d.split]
			expand(i, d.split, offX, offY)
			expand(i, d.rightIdx, offX, offY+lh)
		}
	}
	expand(ti, tj, 0, 0)

	return counts, V[ti][tj], places, waste, true
}

// distinctDims returns the distinct scaled widths (width=true) or heights of the
// options.
func distinctDims(opts []gkOption, width bool) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, o := range opts {
		d := o.h
		if width {
			d = o.w
		}
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			out = append(out, d)
		}
	}
	return out
}

// subsetSums returns every value in [0, cap] reachable as an unbounded sum of the
// given (positive) dimensions — the normal-pattern discretisation of one axis.
func subsetSums(dims []int, cap int) []int {
	reach := make([]bool, cap+1)
	reach[0] = true
	for _, d := range dims {
		if d <= 0 || d > cap {
			continue
		}
		for x := d; x <= cap; x++ {
			if reach[x-d] {
				reach[x] = true
			}
		}
	}
	out := make([]int, 0, 16)
	for x := 0; x <= cap; x++ {
		if reach[x] {
			out = append(out, x)
		}
	}
	return out
}

// floorIndex returns the index of the largest element of the sorted set that is ≤ v
// (set[0] is always 0, so the result is always valid).
func floorIndex(set []int, v int) int {
	i := sort.SearchInts(set, v+1) - 1
	if i < 0 {
		i = 0
	}
	return i
}
