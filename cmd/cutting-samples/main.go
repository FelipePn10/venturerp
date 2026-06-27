// Command cutting-samples runs a batch of representative cutting simulations through
// the real optimisers and writes their cutting maps as SVG, DXF and PDF so you can
// open the files and SEE that nesting works — 1D bars, 2D steel sheets, MDF panels
// (guillotine) and true-shape laser/plasma parts (with free rotation and real
// contours). It uses the domain optimisers and renderer directly; no API or database
// is required.
//
//	go run ./cmd/cutting-samples              # writes to ./cutting-samples/
//	go run ./cmd/cutting-samples -out /tmp/x  # custom output directory
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
)

type sim struct {
	name    string
	cutType entity.CutType
	params  service.CutParams
	demand  []service.DemandPiece
	stock   []service.StockPiece
}

func rectPoly(w, h float64) []service.Point {
	return []service.Point{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h}}
}

// lNotch is an L-shaped contour: a square of side a with a b-deep notch at top-right.
func lNotch(a, b float64) []service.Point {
	return []service.Point{{X: 0, Y: 0}, {X: a, Y: 0}, {X: a, Y: a - b}, {X: a - b, Y: a - b}, {X: a - b, Y: a}, {X: 0, Y: a}}
}

// The demands below are realistic production BATCHES (not a handful of pieces): a
// shop cuts dozens of parts that fill several bars/sheets, so the partly-used last
// stock piece barely affects the totals. Tiny batches look "low utilisation" only
// because their one leftover sheet is mostly a reusable remnant, not scrap.
func sims() []sim {
	return []sim{
		{
			// 1D column generation on heterogeneous bars → ~94% utilisation.
			name: "1d-aco-serralheria", cutType: entity.CutTypeLinear1D,
			params: service.CutParams{Kerf: 3, Trim: 0, MinRemnant: 300},
			demand: []service.DemandPiece{
				{PartID: 1, Label: "Montante 1500", Length: 1500, Qty: 14},
				{PartID: 2, Label: "Travessa 2200", Length: 2200, Qty: 10},
				{PartID: 3, Label: "Diagonal 1200", Length: 1200, Qty: 16},
				{PartID: 4, Label: "Trava 850", Length: 850, Qty: 20},
			},
			stock: []service.StockPiece{
				{StockID: 71, Length: 4300, Qty: 1, IsRemnant: true, Priority: 0},
				{StockID: 70, Length: 6000, Qty: 30, Priority: 10},
			},
		},
		{
			// 2D guillotine, steel sheet (funilaria) → ~87% utilisation.
			name: "2d-aco-funilaria", cutType: entity.CutTypeGuillotine2D,
			params: service.CutParams{Kerf: 4, Trim: 8, MinRemnant: 200},
			demand: []service.DemandPiece{
				{PartID: 1, Label: "Tampa 600x400", Width: 600, Height: 400, Qty: 18, AllowRotation: true},
				{PartID: 2, Label: "Lateral 800x300", Width: 800, Height: 300, Qty: 15, AllowRotation: true},
				{PartID: 3, Label: "Reforco 300x300", Width: 300, Height: 300, Qty: 24, AllowRotation: true},
				{PartID: 4, Label: "Porta 1200x500", Width: 1200, Height: 500, Qty: 9, AllowRotation: true},
			},
			stock: []service.StockPiece{{StockID: 80, Width: 2440, Height: 1220, Qty: 12}},
		},
		{
			// 2D MDF (moveleiro) with grain on the gables/doors, sheet-friendly sizes
			// → ~95% utilisation even with the no-rotation grain constraint.
			name: "2d-mdf-moveleiro-veio", cutType: entity.CutTypeGuillotine2D,
			params: service.CutParams{Kerf: 4, Trim: 10, MinRemnant: 200},
			demand: []service.DemandPiece{
				{PartID: 1, Label: "Lateral 680x900 (veio)", Width: 680, Height: 900, Grain: service.GrainLength, Qty: 16},
				{PartID: 2, Label: "Prateleira 900x450", Width: 900, Height: 450, Qty: 20, AllowRotation: true},
				{PartID: 3, Label: "Fundo 680x450", Width: 680, Height: 450, Qty: 20, AllowRotation: true},
			},
			stock: []service.StockPiece{{StockID: 90, Width: 2750, Height: 1830, Qty: 10}},
		},
		{
			// Showcase: three parts tile a sheet exactly → 100% (column generation finds
			// the perfect guillotine layout a one-shot heuristic would miss).
			name: "2d-guillotine-tiling-100pct", cutType: entity.CutTypeGuillotine2D,
			params: service.CutParams{},
			demand: []service.DemandPiece{
				{PartID: 1, Label: "A 70x70", Width: 70, Height: 70, Qty: 1},
				{PartID: 2, Label: "B 30x70", Width: 30, Height: 70, Qty: 1},
				{PartID: 3, Label: "C 100x30", Width: 100, Height: 30, Qty: 1},
			},
			stock: []service.StockPiece{{StockID: 100, Width: 100, Height: 100, Qty: 2}},
		},
		{
			// True-shape laser/plasma: real L-contours interlocking with rectangles
			// (free rotation) → ~86% on irregular parts.
			name: "trueshape-laser-flanges", cutType: entity.CutTypeTrueShape2D,
			params: service.CutParams{Kerf: 1},
			demand: []service.DemandPiece{
				{PartID: 1, Label: "Flange L grande", Polygon: lNotch(400, 150), Qty: 8, AllowRotation: true},
				{PartID: 2, Label: "Flange L media", Polygon: lNotch(260, 90), Qty: 12, AllowRotation: true},
				{PartID: 3, Label: "Cartela 200x200", Polygon: rectPoly(200, 200), Qty: 12, AllowRotation: true},
			},
			stock: []service.StockPiece{{StockID: 110, Width: 1500, Height: 1000, Qty: 5}},
		},
		{
			// FEATURE demo (not a utilisation showcase): a 130×10 bar fits a 100×100
			// sheet ONLY when rotated ~45° — proves free rotation. Low utilisation here
			// is expected: the demand is just a couple of tiny parts.
			name: "trueshape-barra-diagonal", cutType: entity.CutTypeTrueShape2D,
			params: service.CutParams{},
			demand: []service.DemandPiece{
				{PartID: 1, Label: "Barra 130x10 (so cabe a 45)", Polygon: rectPoly(130, 10), Qty: 1, AllowRotation: true},
				{PartID: 2, Label: "Quina L", Polygon: lNotch(40, 14), Qty: 3, AllowRotation: true},
			},
			stock: []service.StockPiece{{StockID: 120, Width: 100, Height: 100, Qty: 3}},
		},
	}
}

func main() {
	out := flag.String("out", "cutting-samples", "output directory for the SVG/DXF/PDF maps")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}

	branding := service.MapBranding{CompanyName: "VentureERP — Demonstração de Corte", GeneratedAt: time.Now()}
	fmt.Printf("Gerando mapas de corte em %q\n\n", *out)
	fmt.Printf("%-30s %-14s %7s %9s %9s %7s\n", "SIMULAÇÃO", "TIPO", "BARR/CH", "APROV.%", "SUCATA%", "N/ALOC")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────")

	for i, s := range sims() {
		opt, err := service.Optimizer(s.cutType)
		if err != nil {
			fmt.Printf("%-32s  ERRO: %v\n", s.name, err)
			continue
		}
		sol, err := opt.Optimize(s.demand, s.stock, s.params)
		if err != nil {
			fmt.Printf("%-32s  ERRO: %v\n", s.name, err)
			continue
		}
		patterns := toPatterns(sol, geometryByPart(s.demand))

		base := fmt.Sprintf("%d_%s", i+1, s.name)
		code := int64(i + 1)
		for _, f := range []service.MapFormat{service.MapSVG, service.MapDXF, service.MapPDF} {
			data, _, err := service.RenderCutMap(code, patterns, f, branding)
			if err != nil {
				fmt.Printf("  render %s: %v\n", f, err)
				continue
			}
			path := filepath.Join(*out, base+"."+string(f))
			if err := os.WriteFile(path, data, 0o644); err != nil {
				fmt.Printf("  write %s: %v\n", path, err)
			}
		}

		fmt.Printf("%-30s %-14s %7d %8.1f%% %8.1f%% %7d\n",
			s.name, s.cutType, sol.StockUsed, sol.Utilization*100, scrapPct(sol, s.params.MinRemnant), unplacedQty(sol))
	}

	fmt.Print(`
APROV.%  = demanda ÷ estoque consumido (inclui a sobra da última barra/chapa).
SUCATA%  = perda REAL — exclui a sobra reaproveitável (≥ sobra mínima), que volta ao
           estoque como retalho. É a métrica que importa para custo.
Aprov. abaixo de ~90% num lote pequeno é a última chapa parcialmente usada (retalho),
não desperdício; em lotes reais o nesting fica em 85–95% (a barra diagonal é só uma
demonstração da rotação livre, com 2 peças minúsculas).

Pronto. Abra os arquivos .svg (navegador), .pdf (visualizador) ou .dxf (CAD).
`)
}

// scrapPct is the real waste fraction: consumed stock minus placed parts minus the
// reusable remnants (leftover ≥ minRemnant, which returns to inventory).
func scrapPct(sol *service.Solution, minRemnant float64) float64 {
	var reusable float64
	for _, pat := range sol.Patterns {
		if pat.StockWidth > 0 { // 2D / true-shape: reusable rectangle when both sides clear the minimum
			if minRemnant > 0 && pat.RemnantWidth >= minRemnant && pat.RemnantHeight >= minRemnant {
				reusable += pat.RemnantWidth * pat.RemnantHeight * float64(pat.Repeat)
			}
		} else if pat.Remnant >= minRemnant { // 1D
			reusable += pat.Remnant * float64(pat.Repeat)
		}
	}
	scrap := sol.TotalStock - sol.TotalDemand - reusable
	if scrap < 0 {
		scrap = 0
	}
	if sol.TotalStock <= 0 {
		return 0
	}
	return scrap / sol.TotalStock * 100
}

func unplacedQty(s *service.Solution) int {
	n := 0
	for _, d := range s.Unplaced {
		n += d.Qty
	}
	return n
}

// geometryByPart returns the JSON polygon of each true-shape part, keyed by PartID,
// so the renderer can draw the real contour at each placement.
func geometryByPart(demand []service.DemandPiece) map[int64]string {
	m := map[int64]string{}
	for _, d := range demand {
		if len(d.Polygon) >= 3 {
			if b, err := json.Marshal(d.Polygon); err == nil {
				m[d.PartID] = string(b)
			}
		}
	}
	return m
}

// toPatterns mirrors the use-case mapping from a solver Solution to persistable
// patterns, additionally attaching the real contour (Outline) to true-shape
// placements so the map draws shapes instead of bounding boxes.
func toPatterns(sol *service.Solution, geom map[int64]string) []*entity.CuttingPattern {
	pats := make([]*entity.CuttingPattern, 0, len(sol.Patterns))
	for i, sp := range sol.Patterns {
		sheetArea := sp.StockWidth * sp.StockHeight
		util := 0.0
		switch {
		case sheetArea > 0:
			util = sp.UsedArea / sheetArea * 100
		case sp.StockLength > 0:
			util = sp.UsedLength / sp.StockLength * 100
		}
		pat := &entity.CuttingPattern{
			Sequence: i + 1, StockLengthMM: sp.StockLength, RepeatCount: sp.Repeat,
			UsedMM: sp.UsedLength, KerfLossMM: sp.KerfLoss, RemnantMM: sp.Remnant,
			UtilizationPct: util, IsRemnant: sp.IsRemnant,
			StockWidthMM: sp.StockWidth, StockHeightMM: sp.StockHeight,
			UsedAreaMM2: sp.UsedArea, RemnantAreaMM2: sp.RemnantArea,
			RemnantWidthMM: sp.RemnantWidth, RemnantHeightMM: sp.RemnantHeight,
		}
		for j, pl := range sp.Placements {
			pid := pl.PartID
			var pref *int64
			if pid > 0 {
				pref = &pid
			}
			place := &entity.PatternPlacement{
				Sequence: j + 1, PartID: pref, Label: pl.Label, LengthMM: pl.Length,
				OffsetMM: pl.Offset, PosXMM: pl.X, PosYMM: pl.Y, WidthMM: pl.W, HeightMM: pl.H,
				Rotated: pl.Rotated, RotationDeg: pl.RotationDeg,
			}
			if g, ok := geom[pl.PartID]; ok {
				if outline, ok := service.OutlineForPlacement(g, pl.RotationDeg); ok {
					place.Outline = outline
				}
			}
			pat.Placements = append(pat.Placements, place)
		}
		pats = append(pats, pat)
	}
	return pats
}
