package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dia(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestInicioDoPeriodo(t *testing.T) {
	quarta := dia("2026-11-04")
	for periodo, esperado := range map[string]string{
		LotePeriodoNenhum: "2026-11-04", LotePeriodoDia: "2026-11-04",
		LotePeriodoSemana: "2026-11-02", LotePeriodoMes: "2026-11-01",
	} {
		if got := InicioDoPeriodo(periodo, quarta).Format("2006-01-02"); got != esperado {
			t.Errorf("%s: %s, esperado %s", periodo, got, esperado)
		}
	}
	if got := InicioDoPeriodo(LotePeriodoSemana, dia("2026-11-08")).Format("2006-01-02"); got != "2026-11-02" {
		t.Errorf("domingo pertence à semana que começou na segunda: %s", got)
	}
}

func TestDividirPorLoteMaximo(t *testing.T) {
	q := decimal.RequireFromString
	casos := []struct {
		qtd, max, mult string
		esperado       []string
	}{
		{"250", "0", "0", []string{"250"}},
		{"250", "100", "0", []string{"100", "100", "50"}},
		{"80", "100", "0", []string{"80"}},
		// Máximo 100 com múltiplo 30: cada ordem fica em 90 (múltiplo de 30).
		{"240", "100", "30", []string{"90", "90", "60"}},
		// Máximo menor que o múltiplo: vale o máximo informado.
		{"50", "20", "30", []string{"20", "20", "10"}},
	}
	for _, c := range casos {
		got := DividirPorLoteMaximo(q(c.qtd), q(c.max), q(c.mult))
		if len(got) != len(c.esperado) {
			t.Fatalf("%s/%s/%s: %v", c.qtd, c.max, c.mult, got)
		}
		soma := decimal.Zero
		for i := range got {
			if !got[i].Equal(q(c.esperado[i])) {
				t.Fatalf("%s/%s/%s: %v, esperado %v", c.qtd, c.max, c.mult, got, c.esperado)
			}
			soma = soma.Add(got[i])
		}
		if !soma.Equal(q(c.qtd)) {
			t.Fatalf("a soma das ordens é a necessidade: %v", got)
		}
	}
}

func TestValidarLoteDaPolitica(t *testing.T) {
	item, classe := int64(1), int64(2)
	semana, errado := LotePeriodoSemana, "QUINZENA"
	max := decimal.NewFromInt(100)
	zero := decimal.Zero
	if msg := (&PoliticaPlanejamento{ClassificationID: &classe, LotePeriodo: &semana}).Validar(); msg != "" {
		t.Fatalf("período na classificação vale: %s", msg)
	}
	if (&PoliticaPlanejamento{ClassificationID: &classe, LoteMaximo: &max}).Validar() == "" {
		t.Fatal("lote máximo é só do item")
	}
	if (&PoliticaPlanejamento{ItemCode: &item, LoteMaximo: &zero}).Validar() == "" {
		t.Fatal("lote máximo zero")
	}
	if (&PoliticaPlanejamento{ItemCode: &item, LotePeriodo: &errado}).Validar() == "" {
		t.Fatal("período inválido")
	}
	if msg := (&PoliticaPlanejamento{ItemCode: &item, LoteMaximo: &max}).Validar(); msg != "" {
		t.Fatalf("só o lote máximo já é política: %s", msg)
	}
}

func TestHerancaDoLote(t *testing.T) {
	item := int64(7)
	mes, semana := LotePeriodoMes, LotePeriodoSemana
	max := decimal.NewFromInt(50)
	empresa := PoliticaEfetiva{LotePeriodo: LotePeriodoNenhum, Origem: OrigemPolitica{LotePeriodo: OrigemEmpresa}}
	porClasse := map[int64][]*PoliticaPlanejamento{item: {{ID: 1, ClassificationCode: "PARAF", ClassificationLevel: 1, LotePeriodo: &mes}}}
	ef := ResolverPoliticas(empresa, nil, porClasse)[item]
	if ef.LotePeriodo != LotePeriodoMes || ef.Origem.LotePeriodo != "CLASSIFICACAO:PARAF" || !ef.LoteMaximo.IsZero() {
		t.Fatalf("da classificação: %+v", ef)
	}
	doItem := map[int64]*PoliticaPlanejamento{item: {ItemCode: &item, LotePeriodo: &semana, LoteMaximo: &max}}
	ef = ResolverPoliticas(empresa, doItem, porClasse)[item]
	if ef.LotePeriodo != LotePeriodoSemana || !ef.LoteMaximo.Equal(max) || ef.Origem.LoteMaximo != OrigemItem {
		t.Fatalf("o item vence: %+v", ef)
	}
}
