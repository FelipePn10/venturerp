package service

import (
	"math"
	"sort"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// trueShapeNester is the registered TRUE_SHAPE_2D strategy. It is a dispatcher:
//
//   - when the demand carries real polygon outlines it uses a SHAPE-AWARE raster
//     nester (an occupancy grid + bottom-left fill with 90° rotations) so pieces
//     tuck into each other's concavities — a native irregular nester, no external
//     dependency;
//   - when parts are plain rectangles (no polygon) it falls back to the bounding-box
//     path (the 2D guillotine), which is exact and faster for that case.
//
// The raster grid is bounded (≤ rasterCap cells per axis) so it stays fast; for
// maximum yield on heavy true-shape jobs the external engine still overrides this.
type trueShapeNester struct{}

func init() { register(trueShapeNester{}) }

func (trueShapeNester) Type() entity.CutType { return entity.CutTypeTrueShape2D }

const rasterCap = 120 // max grid cells per axis

func (trueShapeNester) Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	hasPolygon := false
	for _, d := range demand {
		if len(d.Polygon) >= 3 {
			hasPolygon = true
			break
		}
	}
	if !hasPolygon {
		return trueShapeBBox{}.Optimize(demand, stock, p)
	}
	return nestMetaheuristic(demand, stock, p)
}

// mask is a part's footprint on the grid: cells[r*w+c] true where the part covers.
type mask struct {
	w, h   int
	cells  []bool
	bboxW  float64 // real bbox dims of this orientation (mm)
	bboxH  float64
	rotDeg float64
}

// rasterSheet is an open sheet with an occupancy grid.
type rasterSheet struct {
	stockID    int64
	width      float64
	height     float64
	trim       float64
	gw, gh     int
	occ        []bool
	isRemnant  bool
	placements []Placement
	usedArea   float64
}

type rasterUnit struct {
	partID  int64
	label   string
	poly    []Point
	w, h    float64
	canRot  bool
	areaApx float64
}

// rasterStock is an available sheet type for the raster nester (package-level so the
// ordered-placement core can be shared by nestRaster and the metaheuristic search).
type rasterStock struct {
	id        int64
	w, h      float64
	remaining int
	isRemnant bool
	priority  int
}

func nestRaster(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	res := pickRes(stock, p.Trim)
	units := expandRasterUnits(demand)
	// Heuristic order: largest bbox-area first.
	sort.SliceStable(units, func(i, j int) bool { return units[i].areaApx > units[j].areaApx })
	masks := make([][]mask, len(units))
	for i := range units {
		masks[i] = buildMasks(units[i], res)
	}
	open, unplaced := placeRasterOrder(units, masks, buildRasterPool(stock), res, p.Trim)
	return buildRasterSolution(open, unplaced), nil
}

// expandRasterUnits flattens demand into one raster unit per required piece (bbox
// derived from the polygon when not given), in input order.
func expandRasterUnits(demand []DemandPiece) []rasterUnit {
	var units []rasterUnit
	for _, d := range demand {
		w, h := d.Width, d.Height
		if (w <= 0 || h <= 0) && len(d.Polygon) >= 3 {
			w, h = PolygonBBox(d.Polygon)
		}
		if w <= 0 || h <= 0 || d.Qty <= 0 {
			continue
		}
		for i := 0; i < d.Qty; i++ {
			units = append(units, rasterUnit{partID: d.PartID, label: d.Label, poly: d.Polygon, w: w, h: h, canRot: d.AllowRotation, areaApx: w * h})
		}
	}
	return units
}

func buildRasterPool(stock []StockPiece) []rasterStock {
	var pool []rasterStock
	for _, s := range stock {
		if s.Width <= 0 || s.Height <= 0 || s.Qty <= 0 {
			continue
		}
		pool = append(pool, rasterStock{id: s.StockID, w: s.Width, h: s.Height, remaining: s.Qty, isRemnant: s.IsRemnant, priority: s.Priority})
	}
	return pool
}

// placeRasterOrder places the units in the GIVEN order (units[k] uses masks[k]),
// opening a new sheet whenever the running sheets cannot hold a piece. It copies the
// stock pool so it is pure and can be re-run for every candidate order the
// metaheuristic explores.
func placeRasterOrder(units []rasterUnit, masks [][]mask, basePool []rasterStock, res, trim float64) ([]*rasterSheet, []DemandPiece) {
	pool := make([]rasterStock, len(basePool))
	copy(pool, basePool)

	var (
		open     []*rasterSheet
		unplaced []DemandPiece
	)
	for k, u := range units {
		ms := masks[k]
		if placeRaster(open, u, ms, res) {
			continue
		}
		// open a new sheet that can physically hold the unit in SOME rotation (the
		// mask footprints already account for free rotation, so checking the bbox of
		// just 0°/90° would wrongly reject a piece that only fits diagonally).
		idx := -1
		for i := range pool {
			st := pool[i]
			if st.remaining <= 0 || !masksFitSheet(ms, st, res, trim) {
				continue
			}
			if idx == -1 {
				idx = i
				continue
			}
			cur := pool[idx]
			if st.priority < cur.priority || (st.priority == cur.priority && st.w*st.h > cur.w*cur.h) {
				idx = i
			}
		}
		if idx == -1 {
			unplaced = appendDemand2D(unplaced, unit2D{partID: u.partID, label: u.label, w: u.w, h: u.h})
			continue
		}
		pool[idx].remaining--
		st := pool[idx]
		sh := newRasterSheet(st.id, st.w, st.h, trim, res, st.isRemnant)
		open = append(open, sh)
		if !placeRaster([]*rasterSheet{sh}, u, ms, res) {
			unplaced = appendDemand2D(unplaced, unit2D{partID: u.partID, label: u.label, w: u.w, h: u.h})
		}
	}

	return open, unplaced
}

func pickRes(stock []StockPiece, trim float64) float64 {
	maxDim := 0.0
	for _, s := range stock {
		maxDim = math.Max(maxDim, math.Max(s.Width, s.Height))
	}
	if maxDim <= 0 {
		return 1
	}
	return math.Max(1, math.Ceil(maxDim/rasterCap))
}

// masksFitSheet reports whether at least one of the unit's rotation masks fits on the
// sheet's usable grid, i.e. the piece can be placed in some orientation.
func masksFitSheet(masks []mask, st rasterStock, res, trim float64) bool {
	gw := int(math.Floor((st.w - trim) / res))
	gh := int(math.Floor((st.h - trim) / res))
	for _, m := range masks {
		if m.w <= gw && m.h <= gh {
			return true
		}
	}
	return false
}

func newRasterSheet(id int64, w, h, trim, res float64, isRemnant bool) *rasterSheet {
	gw := int(math.Floor((w - trim) / res))
	gh := int(math.Floor((h - trim) / res))
	if gw < 0 {
		gw = 0
	}
	if gh < 0 {
		gh = 0
	}
	return &rasterSheet{stockID: id, width: w, height: h, trim: trim, gw: gw, gh: gh, occ: make([]bool, gw*gh), isRemnant: isRemnant}
}

// rasterAngles is the rotation set tried for a rotatable true-shape piece. Going
// beyond 90° steps (FASE 7 — free rotation) lets pieces nest diagonally: a part too
// long to fit axis-aligned can still fit at 45°, and irregular outlines interlock at
// angles a 90°-only nester cannot reach. Each angle is rasterised exactly (the
// polygon is rotated, then sampled onto the grid), so collision stays grid-sound.
var rasterAngles = []float64{0, 45, 90, 135, 180, 225, 270, 315}

// buildMasks rasterises the unit's polygon (or its bounding rectangle) at each allowed
// rotation, deduping identical footprints.
func buildMasks(u rasterUnit, res float64) []mask {
	if !u.canRot {
		return []mask{rasterizeAngle(u, res, 0)}
	}
	var out []mask
	for _, deg := range rasterAngles {
		m := rasterizeAngle(u, res, deg)
		dup := false
		for _, e := range out {
			if e.w == m.w && e.h == m.h && equalCells(e.cells, m.cells) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, m)
		}
	}
	return out
}

// rasterizeAngle rotates the unit's outline by angleDeg and rasterises it (the polygon
// is normalised to the origin first), returning the occupancy mask plus the rotated
// bounding-box dimensions used for the placement record.
func rasterizeAngle(u rasterUnit, res, angleDeg float64) mask {
	poly := u.poly
	if len(poly) < 3 {
		poly = []Point{{0, 0}, {u.w, 0}, {u.w, u.h}, {0, u.h}}
	}
	rp := rotatePoly(poly, angleDeg)
	_, _, bw, bh := polyBounds(rp)

	w := int(math.Ceil(bw / res))
	h := int(math.Ceil(bh / res))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m := mask{w: w, h: h, cells: make([]bool, w*h), bboxW: bw, bboxH: bh, rotDeg: angleDeg}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			cx := (float64(c) + 0.5) * res
			cy := (float64(r) + 0.5) * res
			if pointInPolygon(cx, cy, rp) {
				m.cells[r*w+c] = true
			}
		}
	}
	return m
}

func equalCells(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// placeRaster tries to bottom-left place the unit (any orientation) into any open
// sheet; on success it marks the grid and records the placement.
func placeRaster(sheets []*rasterSheet, u rasterUnit, masks []mask, res float64) bool {
	for _, sh := range sheets {
		for _, m := range masks {
			if m.w > sh.gw || m.h > sh.gh {
				continue
			}
			for r := 0; r+m.h <= sh.gh; r++ {
				for c := 0; c+m.w <= sh.gw; c++ {
					if fits(sh, m, r, c) {
						stamp(sh, m, r, c)
						sh.placements = append(sh.placements, Placement{
							PartID: u.partID, Label: u.label,
							X: sh.trim + float64(c)*res, Y: sh.trim + float64(r)*res,
							W: m.bboxW, H: m.bboxH, Rotated: m.rotDeg != 0, RotationDeg: m.rotDeg,
						})
						sh.usedArea += u.w * u.h
						return true
					}
				}
			}
		}
	}
	return false
}

func fits(sh *rasterSheet, m mask, r, c int) bool {
	for mr := 0; mr < m.h; mr++ {
		for mc := 0; mc < m.w; mc++ {
			if !m.cells[mr*m.w+mc] {
				continue
			}
			if sh.occ[(r+mr)*sh.gw+(c+mc)] {
				return false
			}
		}
	}
	return true
}

func stamp(sh *rasterSheet, m mask, r, c int) {
	for mr := 0; mr < m.h; mr++ {
		for mc := 0; mc < m.w; mc++ {
			if m.cells[mr*m.w+mc] {
				sh.occ[(r+mr)*sh.gw+(c+mc)] = true
			}
		}
	}
}

func buildRasterSolution(open []*rasterSheet, unplaced []DemandPiece) *Solution {
	sol := &Solution{Unplaced: unplaced}
	var totalDemand, totalStock float64
	for i, sh := range open {
		pat := Pattern{
			StockID: sh.stockID, IsRemnant: sh.isRemnant, Repeat: 1,
			StockWidth: sh.width, StockHeight: sh.height, UsedArea: sh.usedArea,
			Placements: sh.placements,
		}
		_ = i
		sol.Patterns = append(sol.Patterns, pat)
		totalDemand += sh.usedArea
		totalStock += sh.width * sh.height
		sol.CutCount += len(sh.placements)
	}
	sol.StockUsed = len(open)
	sol.TotalDemand = totalDemand
	sol.TotalStock = totalStock
	if totalStock > eps {
		sol.Utilization = totalDemand / totalStock
	}
	return sol
}

// pointInPolygon is the classic ray-casting test.
func pointInPolygon(x, y float64, poly []Point) bool {
	in := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi := poly[i].X, poly[i].Y
		xj, yj := poly[j].X, poly[j].Y
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
	}
	return in
}
