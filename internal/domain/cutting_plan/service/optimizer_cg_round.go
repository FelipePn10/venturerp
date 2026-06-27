package service

// coverPick is one decision of the integer rounding: run `copies` whole bars/sheets
// of column `col`, placing `place[i]` units of item i on each (place equals the
// pattern composition for a full run, or a trimmed subset when the pattern would
// otherwise over-produce the last few pieces).
type coverPick struct {
	col    int
	copies int
	place  []int
}

// roundCover turns the (fractional) LP column pool into a whole-bar/whole-sheet plan
// with a greedy set-cover: at each step it runs the pattern with the best COVERAGE
// PER UNIT COST — the area (2D) or length (1D) of still-needed pieces it places,
// divided by the stock it consumes — never producing more than the outstanding
// demand. Seeding the pool with granular single-part patterns (see seed*Columns)
// means this also makes the right STOCK-SELECTION choice: a lone small part is cut
// from the cheapest sheet that holds it, not from a full expensive one, which the
// pure LP relaxation cannot tell apart by area alone.
//
// itemSize carries each item's length (1D) or area (2D) so coverage is measured in
// the same units the objective minimises. It returns the chosen picks and the demand
// / stock left for the heuristic residual pass.
func roundCover(columns []cgColumn, bounds, stockRemaining []int, itemSize []float64) (picks []coverPick, remDemand, remStock []int) {
	m := len(bounds)
	remDemand = append([]int(nil), bounds...)
	remStock = append([]int(nil), stockRemaining...)

	for {
		bestIdx, bestCopies := -1, 0
		var bestPlace []int
		bestScore := 0.0

		for j, col := range columns {
			if remStock[col.stock] <= 0 || col.cost <= 0 {
				continue
			}
			lim := remStock[col.stock]
			covers := false
			for i, c := range col.comp {
				if c <= 0 {
					continue
				}
				if remDemand[i] > 0 {
					covers = true
				}
				if l := remDemand[i] / c; l < lim {
					lim = l
				}
			}
			if !covers {
				continue
			}

			place := make([]int, m)
			copies := 1
			var coverAmt float64
			if lim >= 1 {
				copies = lim
				for i, c := range col.comp {
					place[i] = c
					coverAmt += float64(c*copies) * itemSize[i]
				}
			} else {
				for i, c := range col.comp { // trim the pattern to the leftover demand
					use := c
					if use > remDemand[i] {
						use = remDemand[i]
					}
					place[i] = use
					coverAmt += float64(use) * itemSize[i]
				}
			}
			if coverAmt <= 0 {
				continue
			}
			if score := coverAmt / (col.cost * float64(copies)); score > bestScore+1e-12 {
				bestScore, bestIdx, bestCopies, bestPlace = score, j, copies, place
			}
		}
		if bestIdx == -1 {
			break
		}

		picks = append(picks, coverPick{col: bestIdx, copies: bestCopies, place: bestPlace})
		col := columns[bestIdx]
		remStock[col.stock] -= bestCopies
		done := true
		for i := 0; i < m; i++ {
			if remDemand[i] -= bestPlace[i] * bestCopies; remDemand[i] < 0 {
				remDemand[i] = 0
			}
			if remDemand[i] > 0 {
				done = false
			}
		}
		if done {
			break
		}
	}
	return picks, remDemand, remStock
}
