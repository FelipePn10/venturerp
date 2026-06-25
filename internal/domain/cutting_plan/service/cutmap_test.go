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

func TestRenderCutMap_BadFormat(t *testing.T) {
	if _, _, err := RenderCutMap(1, samplefPatterns(), "png", MapBranding{}); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
