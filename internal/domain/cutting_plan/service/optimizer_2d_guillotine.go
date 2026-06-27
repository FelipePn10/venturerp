package service

import (
	"errors"
	"fmt"
	"sort"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// optimizer2DGuillotine nests rectangular parts (chapas / painéis MDF) into
// rectangular sheets using a free-rectangle guillotine heuristic:
//
//   - parts are placed largest-area first;
//   - each part takes the open free rectangle that wastes the least area
//     (best-area-fit), trying the rotated orientation only when allowed (no fixed
//     grain) — so a visible wood/laminate grain stays aligned;
//   - placing a part splits the used free rectangle into two children with a
//     GUILLOTINE cut (edge-to-edge), the saw kerf removed at the cut, choosing the
//     split axis that leaves the larger usable child;
//   - if no open sheet fits, a new sheet is opened (remnants first via Priority,
//     then the largest sheet), keeping every layout guillotine-compatible.
//
// This is a fast, shop-acceptable heuristic, registered behind the same
// CuttingOptimizer contract so callers stay dimension-agnostic.
// optimizer2DGuillotine is no longer registered directly: the 2D column-generation
// engine (optimizer2DCG) owns the GUILLOTINE_2D slot and uses this free-rectangle
// heuristic both as a guaranteed fallback and to finish the integer residual its LP
// relaxation leaves behind.
type optimizer2DGuillotine struct{}

func (optimizer2DGuillotine) Type() entity.CutType { return entity.CutTypeGuillotine2D }

type freeRect struct{ x, y, w, h float64 }

type sheet struct {
	stockID    int64
	width      float64 // full sheet
	height     float64
	isRemnant  bool
	free       []freeRect
	placements []Placement
	usedArea   float64
}

type stock2D struct {
	id        int64
	width     float64
	height    float64
	usableW   float64
	usableH   float64
	trim      float64
	remaining int
	isRemnant bool
	priority  int
}

type unit2D struct {
	partID    int64
	label     string
	w, h      float64
	canRotate bool
}

func (optimizer2DGuillotine) Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	open, unplaced, err := nest2DGuillotine(demand, stock, p)
	if err != nil {
		return nil, err
	}
	return buildSolution2D(open, unplaced, p), nil
}

// nest2DGuillotine is the shared free-rectangle guillotine nesting core. It returns
// the opened sheets plus the pieces that fit nowhere, so both the heuristic's
// Optimize and the column-generation engine's integer-residual pass agree on
// placement geometry, kerf accounting and remnant preference.
func nest2DGuillotine(demand []DemandPiece, stock []StockPiece, p CutParams) ([]*sheet, []DemandPiece, error) {
	if p.Kerf < 0 || p.Trim < 0 || p.MinRemnant < 0 {
		return nil, nil, errors.New("kerf, trim and min_remnant cannot be negative")
	}

	var units []unit2D
	for _, d := range demand {
		if d.Width <= 0 || d.Height <= 0 {
			return nil, nil, fmt.Errorf("2D demand %q needs positive width and height", d.Label)
		}
		if d.Qty <= 0 {
			continue
		}
		canRotate := d.AllowRotation && (d.Grain == "" || d.Grain == GrainNone)
		for i := 0; i < d.Qty; i++ {
			units = append(units, unit2D{partID: d.PartID, label: d.Label, w: d.Width, h: d.Height, canRotate: canRotate})
		}
	}
	// Largest area first; ties by longest side.
	sort.SliceStable(units, func(i, j int) bool {
		ai, aj := units[i].w*units[i].h, units[j].w*units[j].h
		if ai != aj {
			return ai > aj
		}
		return maxf(units[i].w, units[i].h) > maxf(units[j].w, units[j].h)
	})

	var pool []stock2D
	for _, s := range stock {
		if s.Width <= 0 || s.Height <= 0 || s.Qty <= 0 {
			continue
		}
		uw, uh := s.Width-p.Trim, s.Height-p.Trim
		if uw <= 0 || uh <= 0 {
			continue
		}
		pool = append(pool, stock2D{
			id: s.StockID, width: s.Width, height: s.Height, usableW: uw, usableH: uh,
			trim: p.Trim, remaining: s.Qty, isRemnant: s.IsRemnant, priority: s.Priority,
		})
	}

	var (
		open     []*sheet
		unplaced []DemandPiece
	)

	for _, u := range units {
		if placeBestFit(open, u, p.Kerf) {
			continue
		}
		if s := openSheet(pool, u); s != nil {
			open = append(open, s)
			if placeBestFit([]*sheet{s}, u, p.Kerf) {
				continue
			}
		}
		unplaced = appendDemand2D(unplaced, u)
	}

	return open, unplaced, nil
}

// placeBestFit finds the open free rectangle that fits `u` with the least wasted
// area (trying rotation when allowed) and places it there, returning false if it
// fits nowhere.
func placeBestFit(sheets []*sheet, u unit2D, kerf float64) bool {
	var (
		bestSheet *sheet
		bestIdx   = -1
		bestPW    float64
		bestPH    float64
		bestRot   bool
		bestScore = -1.0
	)
	for _, s := range sheets {
		for i, fr := range s.free {
			for _, o := range orientations(u) {
				if o.w <= fr.w+eps && o.h <= fr.h+eps {
					score := fr.w*fr.h - o.w*o.h // best area fit
					if bestScore < 0 || score < bestScore {
						bestSheet, bestIdx, bestPW, bestPH, bestRot, bestScore = s, i, o.w, o.h, o.rotated, score
					}
				}
			}
		}
	}
	if bestSheet == nil {
		return false
	}
	placeInRect(bestSheet, bestIdx, bestPW, bestPH, bestRot, u, kerf)
	return true
}

type orient struct {
	w, h    float64
	rotated bool
}

func orientations(u unit2D) []orient {
	out := []orient{{u.w, u.h, false}}
	if u.canRotate && u.w != u.h {
		out = append(out, orient{u.h, u.w, true})
	}
	return out
}

// placeInRect places the part at the free rectangle's corner and guillotine-splits
// the remainder into two children, charging kerf at the cut.
func placeInRect(s *sheet, idx int, pw, ph float64, rotated bool, u unit2D, kerf float64) {
	fr := s.free[idx]
	s.placements = append(s.placements, Placement{
		PartID: u.partID, Label: u.label, X: fr.x, Y: fr.y, W: pw, H: ph, Rotated: rotated,
	})
	s.usedArea += pw * ph

	leftoverW := fr.w - pw - kerf
	leftoverH := fr.h - ph - kerf

	// Remove the consumed rectangle.
	s.free = append(s.free[:idx], s.free[idx+1:]...)

	// Guillotine split: cut along the axis that leaves the larger child usable.
	// Shorter-leftover-axis rule keeps the bigger offcut intact.
	if leftoverW < leftoverH {
		// Horizontal cut first: right child limited to the part height.
		addFree(s, freeRect{fr.x + pw + kerf, fr.y, leftoverW, ph})
		addFree(s, freeRect{fr.x, fr.y + ph + kerf, fr.w, leftoverH})
	} else {
		// Vertical cut first: top child limited to the part width.
		addFree(s, freeRect{fr.x + pw + kerf, fr.y, leftoverW, fr.h})
		addFree(s, freeRect{fr.x, fr.y + ph + kerf, pw, leftoverH})
	}
}

func addFree(s *sheet, r freeRect) {
	if r.w > eps && r.h > eps {
		s.free = append(s.free, r)
	}
}

// openSheet pops a stock sheet that can hold `u` in some orientation: lowest
// Priority first (remnants ahead of full sheets), then the largest sheet.
func openSheet(pool []stock2D, u unit2D) *sheet {
	idx := -1
	for i := range pool {
		st := pool[i]
		if st.remaining <= 0 || !sheetFits(st, u) {
			continue
		}
		if idx == -1 {
			idx = i
			continue
		}
		cur := pool[idx]
		if st.priority < cur.priority ||
			(st.priority == cur.priority && st.usableW*st.usableH > cur.usableW*cur.usableH) {
			idx = i
		}
	}
	if idx == -1 {
		return nil
	}
	pool[idx].remaining--
	st := pool[idx]
	return &sheet{
		stockID:   st.id,
		width:     st.width,
		height:    st.height,
		isRemnant: st.isRemnant,
		free:      []freeRect{{x: st.trim, y: st.trim, w: st.usableW, h: st.usableH}},
	}
}

func sheetFits(st stock2D, u unit2D) bool {
	if u.w <= st.usableW+eps && u.h <= st.usableH+eps {
		return true
	}
	if u.canRotate && u.h <= st.usableW+eps && u.w <= st.usableH+eps {
		return true
	}
	return false
}

func appendDemand2D(dst []DemandPiece, u unit2D) []DemandPiece {
	for i := range dst {
		if dst[i].PartID == u.partID && dst[i].Label == u.label &&
			abs(dst[i].Width-u.w) < eps && abs(dst[i].Height-u.h) < eps {
			dst[i].Qty++
			return dst
		}
	}
	return append(dst, DemandPiece{PartID: u.partID, Label: u.label, Width: u.w, Height: u.h, Qty: 1})
}

// buildSolution2D groups identical sheet layouts into patterns and rolls up metrics.
func buildSolution2D(open []*sheet, unplaced []DemandPiece, p CutParams) *Solution {
	type group struct {
		pattern Pattern
		count   int
	}
	groups := map[string]*group{}
	var order []string

	var totalDemand, totalStock float64
	var cutCount int

	for _, s := range open {
		sheetArea := s.width * s.height
		var freeArea, biggestW, biggestH, biggestArea float64
		for _, fr := range s.free {
			freeArea += fr.w * fr.h
			if a := fr.w * fr.h; a > biggestArea {
				biggestArea, biggestW, biggestH = a, fr.w, fr.h
			}
		}

		totalDemand += s.usedArea
		totalStock += sheetArea
		cutCount += len(s.placements)

		key := sheetSignature(s)
		if g, ok := groups[key]; ok {
			g.count++
			continue
		}
		pat := Pattern{
			StockID:       s.stockID,
			IsRemnant:     s.isRemnant,
			StockWidth:    s.width,
			StockHeight:   s.height,
			Placements:    append([]Placement(nil), s.placements...),
			UsedArea:      s.usedArea,
			RemnantArea:   freeArea,
			RemnantWidth:  biggestW,
			RemnantHeight: biggestH,
		}
		groups[key] = &group{pattern: pat, count: 1}
		order = append(order, key)
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

func sheetSignature(s *sheet) string {
	type box struct{ w, h, x, y float64 }
	bs := make([]box, len(s.placements))
	for i, pl := range s.placements {
		bs[i] = box{pl.W, pl.H, pl.X, pl.Y}
	}
	sort.Slice(bs, func(i, j int) bool {
		if bs[i].x != bs[j].x {
			return bs[i].x < bs[j].x
		}
		if bs[i].y != bs[j].y {
			return bs[i].y < bs[j].y
		}
		return bs[i].w < bs[j].w
	})
	sig := fmt.Sprintf("S%.3fx%.3f|", s.width, s.height)
	for _, b := range bs {
		sig += fmt.Sprintf("%.2f,%.2f@%.2f,%.2f;", b.w, b.h, b.x, b.y)
	}
	return sig
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
