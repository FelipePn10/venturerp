package ciap

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

var d = decimal.RequireFromString

func dia(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func TestParcelasDoBem(t *testing.T) {
	b := Bem{Codigo: "B1", DataEntrada: dia("2026-03-15"), ICMSOp: d("1000"), NumeroParcelas: 48}
	if b.Parcela(2026, 2) != 0 || b.Parcela(2026, 3) != 1 || b.Parcela(2026, 4) != 2 || b.Parcela(2030, 2) != 48 || b.Parcela(2030, 3) != 0 {
		t.Fatalf("parcelas: %d %d %d %d %d", b.Parcela(2026, 2), b.Parcela(2026, 3), b.Parcela(2026, 4), b.Parcela(2030, 2), b.Parcela(2030, 3))
	}
	// 1000 / 48 = 20,8333… → 20,83 (truncado: o PVA não aceita parcela maior).
	if !b.ValorParcela().Equal(d("20.83")) {
		t.Fatalf("parcela %s", b.ValorParcela())
	}
}

func TestApurarMes(t *testing.T) {
	baixa := dia("2026-10-20")
	bens := []Bem{
		{ID: 1, Codigo: "NOVO", DataEntrada: dia("2026-10-05"), ICMSOp: d("480"), NumeroParcelas: 48},                   // IM, parcela 1
		{ID: 2, Codigo: "ANTIGO", DataEntrada: dia("2025-01-10"), ICMSOp: d("960"), ICMSST: d("0"), NumeroParcelas: 48}, // SI, parcela 22
		{ID: 3, Codigo: "ULTIMA", DataEntrada: dia("2022-11-02"), ICMSOp: d("4800"), NumeroParcelas: 48},                // SI parcela 48 + BA
		{ID: 4, Codigo: "VENDIDO", DataEntrada: dia("2024-06-01"), ICMSOp: d("100"), NumeroParcelas: 48, Baixado: true, DataBaixa: &baixa, TipoBaixa: MovAlienacao},
		{ID: 5, Codigo: "ACABOU", DataEntrada: dia("2020-01-01"), ICMSOp: d("100"), NumeroParcelas: 48}, // fora
		{ID: 6, Codigo: "FUTURO", DataEntrada: dia("2026-11-01"), ICMSOp: d("100"), NumeroParcelas: 48}, // fora
	}
	a, err := Apurar(bens, 2026, 10, d("80000"), d("100000"))
	if err != nil {
		t.Fatal(err)
	}
	tipos := map[string][]string{}
	for _, m := range a.Movimentos {
		tipos[m.Bem.Codigo] = append(tipos[m.Bem.Codigo], m.TipoMov)
	}
	if len(tipos["NOVO"]) != 1 || tipos["NOVO"][0] != "IM" || tipos["ANTIGO"][0] != "SI" ||
		len(tipos["ULTIMA"]) != 2 || tipos["ULTIMA"][1] != "BA" || len(tipos["VENDIDO"]) != 2 || tipos["VENDIDO"][1] != "AT" ||
		tipos["ACABOU"] != nil || tipos["FUTURO"] != nil {
		t.Fatalf("movimentos: %v", tipos)
	}
	// Parcelas: 10 + 20 + 100 = 130; índice 0,8 → 104,00 apropriado.
	if !a.SomaParcelas.Equal(d("130")) || !a.Indice.Equal(d("0.8")) || !a.ICMSApropriado.Equal(d("104")) {
		t.Fatalf("G110: soma %s índice %s apropriado %s", a.SomaParcelas, a.Indice, a.ICMSApropriado)
	}
	// Saldo inicial: os que entraram antes do mês (ANTIGO, ULTIMA, VENDIDO).
	if !a.SaldoInicial.Equal(d("5860")) {
		t.Fatalf("saldo inicial %s", a.SaldoInicial)
	}
	if _, err := Apurar(bens, 2026, 10, d("10"), d("5")); err == nil {
		t.Fatal("tributadas acima do total")
	}
	sem, _ := Apurar(bens, 2026, 10, decimal.Zero, decimal.Zero)
	if !sem.ICMSApropriado.IsZero() {
		t.Fatal("sem saídas, sem crédito")
	}
}
