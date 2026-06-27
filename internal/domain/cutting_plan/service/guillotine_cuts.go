package service

import "sort"

// CutBox is one placed part on a sheet (absolute mm coordinates) used to derive the
// guillotine cut sequence.
type CutBox struct {
	X, Y, W, H float64
	Label      string
}

// GuillotineCut is one full edge-to-edge straight cut of the cutting program — what a
// panel saw (seccionadora) actually executes. Cuts are listed in execution order;
// Level 0 is the primary (head) cut of the sheet, deeper levels are the sub-cuts of
// each resulting panel. Successive parts sharing a cut line is the natural "common
// cut" that saves kerf — it is a single GuillotineCut here.
type GuillotineCut struct {
	Sequence   int
	Level      int
	Axis       string  // "VERTICAL" (a constant-X line) | "HORIZONTAL" (constant-Y)
	PositionMM float64 // the cut-line coordinate on the sheet
	FromMM     float64 // span start along the cut
	ToMM       float64 // span end along the cut
}

// GuillotineCutPlan derives the ordered guillotine cut sequence that separates the
// placed parts of one sheet. It recursively finds a straight cut spanning the whole
// current panel that splits it in two without crossing any part, then recurses on each
// side. Returns nil if the layout is not guillotine-separable (the caller then falls
// back to the raw placement order).
func GuillotineCutPlan(boxes []CutBox, sheetW, sheetH float64) []GuillotineCut {
	var cuts []GuillotineCut
	seq := 0
	if !cutRegion(panel{0, 0, sheetW, sheetH}, boxes, 0, &seq, &cuts) {
		return nil
	}
	return cuts
}

type panel struct{ x, y, w, h float64 }

func cutRegion(r panel, boxes []CutBox, level int, seq *int, cuts *[]GuillotineCut) bool {
	if len(boxes) <= 1 {
		return true // a single part (or empty waste) — nothing left to separate
	}
	if p, left, right, ok := splitAxis(boxes, r.x, r.x+r.w, true); ok {
		*seq++
		*cuts = append(*cuts, GuillotineCut{Sequence: *seq, Level: level, Axis: "VERTICAL", PositionMM: p, FromMM: r.y, ToMM: r.y + r.h})
		return cutRegion(panel{r.x, r.y, p - r.x, r.h}, left, level+1, seq, cuts) &&
			cutRegion(panel{p, r.y, r.x + r.w - p, r.h}, right, level+1, seq, cuts)
	}
	if p, bottom, top, ok := splitAxis(boxes, r.y, r.y+r.h, false); ok {
		*seq++
		*cuts = append(*cuts, GuillotineCut{Sequence: *seq, Level: level, Axis: "HORIZONTAL", PositionMM: p, FromMM: r.x, ToMM: r.x + r.w})
		return cutRegion(panel{r.x, r.y, r.w, p - r.y}, bottom, level+1, seq, cuts) &&
			cutRegion(panel{r.x, p, r.w, r.y + r.h - p}, top, level+1, seq, cuts)
	}
	return false
}

// splitAxis finds a cut line (along X when vertical, else Y) inside (lo,hi) that no box
// straddles and that puts at least one box on each side. Candidate lines are the boxes'
// far edges; the first valid one is taken (leftmost/bottommost = the head cut).
func splitAxis(boxes []CutBox, lo, hi float64, vertical bool) (pos float64, a, b []CutBox, ok bool) {
	edges := make([]float64, 0, len(boxes))
	for _, bx := range boxes {
		if vertical {
			edges = append(edges, bx.X+bx.W)
		} else {
			edges = append(edges, bx.Y+bx.H)
		}
	}
	sort.Float64s(edges)

	for i, p := range edges {
		if p <= lo+eps || p >= hi-eps {
			continue
		}
		if i > 0 && edges[i-1] >= p-eps {
			continue // dedupe
		}
		straddle := false
		for _, bx := range boxes {
			s, e := bx.X, bx.X+bx.W
			if !vertical {
				s, e = bx.Y, bx.Y+bx.H
			}
			if s < p-eps && e > p+eps {
				straddle = true
				break
			}
		}
		if straddle {
			continue
		}
		var lhs, rhs []CutBox
		for _, bx := range boxes {
			e := bx.X + bx.W
			if !vertical {
				e = bx.Y + bx.H
			}
			if e <= p+eps {
				lhs = append(lhs, bx)
			} else {
				rhs = append(rhs, bx)
			}
		}
		if len(lhs) > 0 && len(rhs) > 0 {
			return p, lhs, rhs, true
		}
	}
	return 0, nil, nil, false
}
