package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

func samplefPatterns() []*entity.CuttingPattern {
	return []*entity.CuttingPattern{
		{ // 2D sheet
			Sequence: 1, RepeatCount: 2, StockWidthMM: 2750, StockHeightMM: 1830, UtilizationPct: 92,
			Placements: []*entity.PatternPlacement{
				{Sequence: 1, Label: "Lateral", PosXMM: 0, PosYMM: 0, WidthMM: 600, HeightMM: 700},
				{Sequence: 2, Label: "Prateleira", PosXMM: 610, PosYMM: 0, WidthMM: 564, HeightMM: 300},
			},
		},
		{ // 1D bar
			Sequence: 2, RepeatCount: 1, StockLengthMM: 6000, UtilizationPct: 80,
			Placements: []*entity.PatternPlacement{
				{Sequence: 1, Label: "Perna", OffsetMM: 0, LengthMM: 720},
			},
		},
	}
}

func TestRenderCutMap_SVG(t *testing.T) {
	data, ct, err := RenderCutMap(900, samplefPatterns(), MapSVG, MapBranding{})
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/svg+xml" {
		t.Fatalf("content type = %s", ct)
	}
	s := string(data)
	if !strings.Contains(s, "<svg") || !strings.Contains(s, "<rect") || !strings.Contains(s, "Lateral") {
		t.Fatalf("SVG missing expected content: %.120s", s)
	}
}

func TestRenderCutMap_DXF(t *testing.T) {
	data, _, err := RenderCutMap(900, samplefPatterns(), MapDXF, MapBranding{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "LWPOLYLINE") || !strings.HasSuffix(strings.TrimSpace(s), "EOF") {
		t.Fatalf("DXF malformed: %.120s", s)
	}
}

func TestRenderCutMap_PDF(t *testing.T) {
	data, ct, err := RenderCutMap(900, samplefPatterns(), MapPDF, MapBranding{CompanyName: "Tecnofer Ltda", BrandColorHex: "#1B3A5B"})
	if err != nil {
		t.Fatal(err)
	}
	if ct != "application/pdf" {
		t.Fatalf("content type = %s", ct)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data, []byte(" re")) || !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatalf("PDF malformed (len %d)", len(data))
	}
}

// truedShapePattern is a true-shape sheet with an L-contour placement (Outline set).
func trueShapePattern() []*entity.CuttingPattern {
	return []*entity.CuttingPattern{{
		Sequence: 1, RepeatCount: 1, StockWidthMM: 1000, StockHeightMM: 1000, UtilizationPct: 70,
		Placements: []*entity.PatternPlacement{{
			Sequence: 1, Label: "Flange L", PosXMM: 50, PosYMM: 50, WidthMM: 400, HeightMM: 400, RotationDeg: 0,
			Outline: [][2]float64{{0, 0}, {400, 0}, {400, 250}, {250, 250}, {250, 400}, {0, 400}},
		}},
	}}
}

// TestRenderCutMap_DrawsTrueShapeContours guards the FASE 7 rendering improvement: a
// placement with an Outline must be drawn as its real polygon (SVG <polygon>, a DXF
// LWPOLYLINE with the polygon's vertex count, a closed PDF path), not a bounding rect.
func TestRenderCutMap_DrawsTrueShapeContours(t *testing.T) {
	pats := trueShapePattern()
	svg, _, _ := RenderCutMap(1, pats, MapSVG, MapBranding{})
	if !strings.Contains(string(svg), "<polygon") {
		t.Fatalf("SVG should draw the contour as a polygon: %.160s", svg)
	}
	dxf, _, _ := RenderCutMap(1, pats, MapDXF, MapBranding{})
	// 6-vertex part contour + the 4-vertex stock rectangle.
	if !strings.Contains(string(dxf), "LWPOLYLINE\n8\nCUT\n90\n6\n") {
		t.Fatalf("DXF should contain a 6-vertex LWPOLYLINE for the contour")
	}
	pdf, _, _ := RenderCutMap(1, pats, MapPDF, MapBranding{})
	if !bytes.Contains(pdf, []byte(" m\n")) || !bytes.Contains(pdf, []byte("h S\n")) {
		t.Fatalf("PDF should draw the contour as a closed path")
	}

	// OutlineForPlacement: geometry rotated 90° swaps the bbox dims.
	out, ok := OutlineForPlacement(`[{"x":0,"y":0},{"x":400,"y":0},{"x":400,"y":200},{"x":0,"y":200}]`, 90)
	if !ok || len(out) != 4 {
		t.Fatalf("OutlineForPlacement failed: ok=%v n=%d", ok, len(out))
	}
}

func TestRenderCutMap_BadFormat(t *testing.T) {
	if _, _, err := RenderCutMap(1, samplefPatterns(), "png", MapBranding{}); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
