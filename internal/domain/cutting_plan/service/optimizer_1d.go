package service

import (
	"errors"
	"fmt"
	"sort"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// eps guards float comparisons (lengths are in millimetres; sub-micron slack).
const eps = 1e-6

// optimizer1D nests linear pieces (bars/profiles/tubes) into heterogeneous stock
// using a Best-Fit Decreasing heuristic:
//
//   - demand pieces are cut in decreasing length order (largest first);
//   - each piece first tries the OPEN stock piece that leaves the tightest gap
//     (best fit), packing small pieces into the slack of already-opened bars;
//   - if none fit, a new stock piece is opened, preferring lower Priority first
//     (so reusable remnants are consumed before full bars) and, within the same
//     priority, the longest available length (packs more per bar → fewer bars).
//
// Kerf is charged between consecutive pieces; Trim is removed from the head of
// each stock piece before the first cut. The leftover after the last cut is a
// remnant, reusable when it reaches CutParams.MinRemnant.
//
// This is a fast, shop-acceptable heuristic. The contract leaves room to swap in
// an exact method (column generation / Gilmore-Gomory) later without touching
// callers.
// optimizer1D is no longer registered directly: the column-generation engine
// (optimizer1DCG) owns the LINEAR_1D slot and uses this Best-Fit Decreasing
// heuristic both as a guaranteed fallback and to mop up the integer residual its
// LP relaxation leaves behind.
type optimizer1D struct{}

func (optimizer1D) Type() entity.CutType { return entity.CutTypeLinear1D }

// bin is one opened stock piece being filled.
type bin struct {
	stockID    int64
	length     float64 // full stock length
	usable     float64 // length - trim
	consumed   float64 // pieces + interleaving kerf placed so far, within usable
	isRemnant  bool
	placements []Placement
}

// stockType is an available length with remaining quantity.
type stockType struct {
	id        int64
	length    float64
	usable    float64
	remaining int
	isRemnant bool
	priority  int
}

func (optimizer1D) Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	open, unplaced, err := nest1D(demand, stock, p)
	if err != nil {
		return nil, err
	}
	return buildSolution(open, unplaced, p), nil
}

// nest1D runs the Best-Fit Decreasing nesting and returns the opened bins plus the
// pieces that fit nowhere. It is the shared core behind both optimizer1D.Optimize
// and the column-generation engine's integer-residual pass, so the two always
// agree on placement geometry, kerf accounting and remnant preference.
func nest1D(demand []DemandPiece, stock []StockPiece, p CutParams) ([]*bin, []DemandPiece, error) {
	if p.Kerf < 0 || p.Trim < 0 || p.MinRemnant < 0 {
		return nil, nil, errors.New("kerf, trim and min_remnant cannot be negative")
	}

	// Expand demand into individual units, then sort by length descending.
	type unit struct {
		partID int64
		label  string
		length float64
	}
	var units []unit
	for _, d := range demand {
		if d.Length <= 0 {
			return nil, nil, fmt.Errorf("demand %q has non-positive length", d.Label)
		}
		if d.Qty <= 0 {
			continue
		}
		for i := 0; i < d.Qty; i++ {
			units = append(units, unit{partID: d.PartID, label: d.Label, length: d.Length})
		}
	}
	sort.SliceStable(units, func(i, j int) bool { return units[i].length > units[j].length })

	// Build the stock pool (usable = length - trim; unusable stock is skipped).
	var pool []stockType
	for _, s := range stock {
		if s.Length <= 0 || s.Qty <= 0 {
			continue
		}
		usable := s.Length - p.Trim
		if usable <= 0 {
			continue
		}
		pool = append(pool, stockType{
			id:        s.StockID,
			length:    s.Length,
			usable:    usable,
			remaining: s.Qty,
			isRemnant: s.IsRemnant,
			priority:  s.Priority,
		})
	}

	var (
		open     []*bin
		unplaced []DemandPiece
	)

	for _, u := range units {
		if b := bestOpenBin(open, u.length, p.Kerf); b != nil {
			placeInBin(b, u.partID, u.label, u.length, p.Trim, p.Kerf)
			continue
		}
		if st := openStock(pool, u.length); st != nil {
			b := &bin{stockID: st.id, length: st.length, usable: st.usable, isRemnant: st.isRemnant}
			placeInBin(b, u.partID, u.label, u.length, p.Trim, p.Kerf)
			open = append(open, b)
			continue
		}
		// No stock — open or in the pool — can hold this length.
		unplaced = appendDemand(unplaced, u.partID, u.label, u.length)
	}

	return open, unplaced, nil
}

// bestOpenBin returns the open bin where `length` fits leaving the smallest
// remaining gap (best fit), or nil if it fits in none.
func bestOpenBin(open []*bin, length, kerf float64) *bin {
	var best *bin
	bestGap := -1.0
	for _, b := range open {
		need := length
		if len(b.placements) > 0 {
			need += kerf
		}
		gap := b.usable - (b.consumed + need)
		if gap < -eps {
			continue
		}
		if best == nil || gap < bestGap {
			best = b
			bestGap = gap
		}
	}
	return best
}

// openStock picks and decrements a stock type that can hold `length` as a first
// cut: lowest Priority first (remnants ahead of bars), then longest usable.
func openStock(pool []stockType, length float64) *stockType {
	idx := -1
	for i := range pool {
		if pool[i].remaining <= 0 || pool[i].usable+eps < length {
			continue
		}
		if idx == -1 {
			idx = i
			continue
		}
		cur, cand := pool[idx], pool[i]
		if cand.priority < cur.priority ||
			(cand.priority == cur.priority && cand.usable > cur.usable) {
			idx = i
		}
	}
	if idx == -1 {
		return nil
	}
	pool[idx].remaining--
	st := pool[idx]
	return &st
}

// placeInBin appends a piece to a bin, charging kerf between pieces and locating
// the cut after the trimmed head.
func placeInBin(b *bin, partID int64, label string, length, trim, kerf float64) {
	var pos float64
	if len(b.placements) == 0 {
		pos = b.consumed
		b.consumed += length
	} else {
		pos = b.consumed + kerf
		b.consumed += kerf + length
	}
	b.placements = append(b.placements, Placement{
		PartID: partID,
		Label:  label,
		Length: length,
		Offset: trim + pos,
	})
}

func appendDemand(dst []DemandPiece, partID int64, label string, length float64) []DemandPiece {
	for i := range dst {
		if dst[i].PartID == partID && dst[i].Label == label && abs(dst[i].Length-length) < eps {
			dst[i].Qty++
			return dst
		}
	}
	return append(dst, DemandPiece{PartID: partID, Label: label, Length: length, Qty: 1})
}

// buildSolution groups identical bin layouts into patterns and rolls up metrics.
func buildSolution(open []*bin, unplaced []DemandPiece, p CutParams) *Solution {
	type group struct {
		pattern Pattern
		count   int
	}
	groups := map[string]*group{}
	var order []string

	var totalDemand, totalStock float64
	var cutCount int

	for _, b := range open {
		var used float64
		for _, pl := range b.placements {
			used += pl.Length
		}
		kerfLoss := 0.0
		if len(b.placements) > 1 {
			kerfLoss = float64(len(b.placements)-1) * p.Kerf
		}
		remnant := b.usable - b.consumed // leftover within the usable region

		totalDemand += used
		totalStock += b.length
		cutCount += len(b.placements)

		key := binSignature(b)
		g, ok := groups[key]
		if !ok {
			pat := Pattern{
				StockID:     b.stockID,
				StockLength: b.length,
				IsRemnant:   b.isRemnant,
				Placements:  append([]Placement(nil), b.placements...),
				UsedLength:  used,
				KerfLoss:    kerfLoss,
				Remnant:     remnant,
			}
			groups[key] = &group{pattern: pat, count: 1}
			order = append(order, key)
		} else {
			g.count++
		}
	}

	patterns := make([]Pattern, 0, len(order))
	for _, key := range order {
		g := groups[key]
		g.pattern.Repeat = g.count
		patterns = append(patterns, g.pattern)
	}

	util := 0.0
	if totalStock > eps {
		util = totalDemand / totalStock
	}

	return &Solution{
		Patterns:    patterns,
		Unplaced:    unplaced,
		TotalDemand: totalDemand,
		TotalStock:  totalStock,
		Utilization: util,
		StockUsed:   len(open),
		CutCount:    cutCount,
	}
}

// binSignature identifies layouts that are identical for grouping: same stock
// length and same multiset of cut lengths (order-independent).
func binSignature(b *bin) string {
	lens := make([]float64, len(b.placements))
	for i, pl := range b.placements {
		lens[i] = pl.Length
	}
	sort.Float64s(lens)
	s := fmt.Sprintf("L%.4f|", b.length)
	for _, l := range lens {
		s += fmt.Sprintf("%.4f,", l)
	}
	return s
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
