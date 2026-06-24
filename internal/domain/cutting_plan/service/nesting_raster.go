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
	return nestRaster(demand, stock, p)
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

func nestRaster(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error) {
	res := pickRes(stock, p.Trim)

	// Expand demand into units, largest bbox-area first.
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
	sort.SliceStable(units, func(i, j int) bool { return units[i].areaApx > units[j].areaApx })

	type stockType struct {
		id        int64
		w, h      float64
		remaining int
		isRemnant bool
		priority  int
	}
	var pool []stockType
	for _, s := range stock {
		if s.Width <= 0 || s.Height <= 0 || s.Qty <= 0 {
			continue
		}
		pool = append(pool, stockType{id: s.StockID, w: s.Width, h: s.Height, remaining: s.Qty, isRemnant: s.IsRemnant, priority: s.Priority})
	}

	var (
		open     []*rasterSheet
		unplaced []DemandPiece
	)

	for _, u := range units {
		masks := buildMasks(u, res)
		if placeRaster(open, u, masks, res) {
			continue
		}
		// open a new sheet that can physically hold the unit's bbox
		idx := -1
		for i := range pool {
			st := pool[i]
			if st.remaining <= 0 || !bboxFits(st.w, st.h, u, p.Trim) {
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
		sh := newRasterSheet(st.id, st.w, st.h, p.Trim, res, st.isRemnant)
		open = append(open, sh)
		if !placeRaster([]*rasterSheet{sh}, u, masks, res) {
			unplaced = appendDemand2D(unplaced, unit2D{partID: u.partID, label: u.label, w: u.w, h: u.h})
		}
	}

	return buildRasterSolution(open, unplaced), nil
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

func bboxFits(sw, sh float64, u rasterUnit, trim float64) bool {
	uw, uh := sw-trim, sh-trim
	if u.w <= uw+eps && u.h <= uh+eps {
		return true
	}
	if u.canRot && u.h <= uw+eps && u.w <= uh+eps {
		return true
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

// buildMasks rasterises the unit's polygon (or rectangle) for each allowed 90°
// rotation, deduping identical masks.
func buildMasks(u rasterUnit, res float64) []mask {
	base := rasterizeUnit(u, res, 0)
	out := []mask{base}
	if !u.canRot {
		return out
	}
	cur := base
	for deg := 90; deg < 360; deg += 90 {
		cur = rotateMask(cur)
		cur.rotDeg = float64(deg)
		// swap bbox dims for 90/270
		if deg == 90 || deg == 270 {
			cur.bboxW, cur.bboxH = u.h, u.w
		} else {
			cur.bboxW, cur.bboxH = u.w, u.h
		}
		dup := false
		for _, m := range out {
			if m.w == cur.w && m.h == cur.h && equalCells(m.cells, cur.cells) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, cur)
		}
	}
	return out
}

func rasterizeUnit(u rasterUnit, res float64, rotDeg float64) mask {
	w := int(math.Ceil(u.w / res))
	h := int(math.Ceil(u.h / res))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m := mask{w: w, h: h, cells: make([]bool, w*h), bboxW: u.w, bboxH: u.h, rotDeg: rotDeg}
	if len(u.poly) >= 3 {
		minX, minY := polyMin(u.poly)
		for r := 0; r < h; r++ {
			for c := 0; c < w; c++ {
				cx := minX + (float64(c)+0.5)*res
				cy := minY + (float64(r)+0.5)*res
				if pointInPolygon(cx, cy, u.poly) {
					m.cells[r*w+c] = true
				}
			}
		}
	} else {
		for i := range m.cells {
			m.cells[i] = true
		}
	}
	return m
}

// rotateMask returns the mask rotated 90° clockwise.
func rotateMask(m mask) mask {
	nw, nh := m.h, m.w
	out := mask{w: nw, h: nh, cells: make([]bool, nw*nh)}
	for r := 0; r < m.h; r++ {
		for c := 0; c < m.w; c++ {
			if m.cells[r*m.w+c] {
				nr, nc := c, m.h-1-r
				out.cells[nr*nw+nc] = true
			}
		}
	}
	return out
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

func polyMin(poly []Point) (minX, minY float64) {
	minX, minY = poly[0].X, poly[0].Y
	for _, p := range poly[1:] {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
	}
	return
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
