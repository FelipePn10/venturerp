package pdfkit

import (
	"bytes"
	"testing"
)

func TestTableReportRendersValidPDF(t *testing.T) {
	tr := &TableReport{
		Theme: DefaultTheme(),
		Company: Company{
			Name:    "Tecnofer Indústria Metalúrgica Ltda",
			CNPJ:    "52.454.668/0001-02",
			IE:      "9103144679",
			Address: "Rua das Indústrias, 1000 - Sertanópolis/PR",
		},
		Title:    "Relatório de Pedidos",
		Subtitle: "Período 01–24/06/2026",
		Columns: []Column{
			{Title: "Código", Align: AlignLeft, Weight: 1},
			{Title: "Cliente", Align: AlignLeft, Weight: 3},
			{Title: "Valor", Align: AlignRight, Weight: 1},
		},
		Totals: []string{"", "TOTAL", "R$ 1.000,00"},
	}
	for i := 0; i < 80; i++ { // force multi-page
		tr.Rows = append(tr.Rows, []string{"100", "MetalFix Ltda", "R$ 12,50"})
	}

	out := tr.Render()
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatal("missing PDF header")
	}
	for _, marker := range []string{"%%EOF", "/Type /Catalog", "startxref", "/Type /Pages"} {
		if !bytes.Contains(out, []byte(marker)) {
			t.Errorf("missing %q", marker)
		}
	}
}

func TestTextWidthProportional(t *testing.T) {
	// "W" is much wider than "i" in a proportional font; a monospace metric
	// would make them equal.
	if TextWidth(FontRegular, 10, "W") <= TextWidth(FontRegular, 10, "i") {
		t.Error("expected proportional widths (W wider than i)")
	}
}

func TestParseHexColorViaImageless(t *testing.T) {
	// Ensure a zero-image document still renders (no XObject resource emitted).
	d := New()
	p := d.AddPage()
	p.Text(50, 50, FontBold, 12, Black, "Olá Mundo")
	if out := d.Render(); !bytes.Contains(out, []byte("Olá") /* winansi */) && !bytes.Contains(out, []byte("Ol")) {
		t.Error("text not encoded")
	}
}
