package entity

import "testing"

func TestLerGS1(t *testing.T) {
	g, ok := LerGS1("(01)07891234567895(17)261231(10)L-77\x1d(37)12")
	if !ok || g.GTIN != "07891234567895" || g.Lote != "L-77" || g.Quantidade == nil || !g.Quantidade.Equal(dd("12")) {
		t.Fatalf("%+v %v", g, ok)
	}
	if g.Validade == nil || g.Validade.Format("2006-01-02") != "2026-12-31" {
		t.Fatalf("validade: %v", g.Validade)
	}
	g, ok = LerGS1("]C1010789123456789510LOTE9")
	if !ok || g.Lote != "LOTE9" {
		t.Fatalf("lote no fim, sem separador: %+v %v", g, ok)
	}
	g, ok = LerGS1("01078912345678951726020010X")
	if !ok || g.Validade == nil || g.Validade.Format("2006-01-02") != "2026-02-28" || g.Lote != "X" {
		t.Fatalf("dia 00 é o último do mês: %+v %v", g, ok)
	}
	for _, s := range []string{"MP-CH-3MM", "7891234567895", "01ABC"} {
		if _, ok := LerGS1(s); ok {
			t.Fatalf("%q não é GS1", s)
		}
	}
}
