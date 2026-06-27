package service

// boundedKnapsack is the pricing subproblem of the 1D column generation: given the
// dual prices of the demand constraints it finds the most valuable cutting pattern
// that fits one stock bar. Formally it maximises Σ valuesᵢ·countᵢ subject to
// Σ weightsᵢ·countᵢ ≤ capacity and 0 ≤ countᵢ ≤ boundsᵢ, with integer counts.
//
// weights, bounds and capacity are non-negative integers (lengths are scaled to
// integers by the caller). Each demand type is expanded into binary-weighted
// pseudo-items (1,2,4,… copies) so a bounded knapsack reduces to a 0/1 knapsack
// solved by dynamic programming — O(pseudoItems · capacity) — with the chosen
// counts reconstructed from a bit-packed decision table to keep memory modest.
func boundedKnapsack(values []float64, weights, bounds []int, capacity int) (counts []int, value float64) {
	n := len(values)
	counts = make([]int, n)
	if capacity <= 0 || n == 0 {
		return counts, 0
	}

	// Binary decomposition into 0/1 pseudo-items.
	type pseudo struct {
		item int
		mult int
		w    int
		v    float64
	}
	var items []pseudo
	for i := 0; i < n; i++ {
		if weights[i] <= 0 || bounds[i] <= 0 || values[i] <= 0 {
			continue // a zero-value or unbounded-weight piece never helps the pattern
		}
		rem := bounds[i]
		for k := 1; rem > 0; k <<= 1 {
			take := k
			if take > rem {
				take = rem
			}
			items = append(items, pseudo{item: i, mult: take, w: weights[i] * take, v: values[i] * float64(take)})
			rem -= take
		}
	}
	if len(items) == 0 {
		return counts, 0
	}

	dp := make([]float64, capacity+1)
	words := capacity/64 + 1
	take := make([][]uint64, len(items)) // take[k] bit c set ⇒ pseudo-item k chosen at capacity c

	for k, it := range items {
		row := make([]uint64, words)
		for c := capacity; c >= it.w; c-- {
			if cand := dp[c-it.w] + it.v; cand > dp[c]+1e-12 {
				dp[c] = cand
				row[c>>6] |= 1 << uint(c&63)
			}
		}
		take[k] = row
	}

	// Reconstruct the counts by walking the decision table backwards.
	c := capacity
	for k := len(items) - 1; k >= 0; k-- {
		if take[k][c>>6]&(1<<uint(c&63)) != 0 {
			counts[items[k].item] += items[k].mult
			c -= items[k].w
		}
	}
	return counts, dp[capacity]
}
