package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

func TestDecideAutoReview(t *testing.T) {
	pol := DefaultCreditPolicy()
	com := func(sug int64, classe RiskClass) CreditSuggestion {
		return CreditSuggestion{SuggestedLimit: dec(sug), RiskClass: classe, HasHistory: true, Score: 700}
	}
	casos := []struct {
		nome   string
		s      CreditSuggestion
		atual  int64
		ultima *LastCreditReview
		quer   int64
		mudou  bool
		contem string
	}{
		{"atraso reduz na hora", func() CreditSuggestion {
			s := com(20000, RiskD)
			s.Factors = []CreditFactor{{Code: "ATRASO_MEDIO", Label: "Atraso médio", Points: 20, MaxPoints: 200, Detail: "27 dias"}}
			return s
		}(), 50000, &LastCreditReview{RiskClass: RiskB}, 20000, true, "atraso médio 27 dias"},
		{"bloqueio zera", CreditSuggestion{HardStop: true, RiskClass: RiskE,
			Factors: []CreditFactor{{Code: "PONTUALIDADE", Label: "Valor pago em dia", Points: 10, MaxPoints: 300, Detail: "13%"},
				{Code: "PROTESTO", Label: "Título em protesto", Blocking: true, Detail: "protesto"}}}, 50000, nil, 0, true, "(título em protesto: protesto)"},
		{"exceção sem piora se mantém", com(30000, RiskB), 50000, &LastCreditReview{RiskClass: RiskB, Exception: true}, 50000, false, "exceção"},
		{"exceção com piora reduz", com(30000, RiskC), 50000, &LastCreditReview{RiskClass: RiskB, Exception: true}, 30000, true, "B → C"},
		{"bom pagador sobe até +50%", com(200000, RiskA), 100000, &LastCreditReview{RiskClass: RiskB}, 150000, true, "+50%"},
		{"sobe até o sugerido", com(120000, RiskA), 100000, nil, 120000, true, "aumentado"},
		{"sem histórico não sobe", CreditSuggestion{SuggestedLimit: dec(40000), RiskClass: RiskC}, 10000, nil, 10000, false, "histórico"},
		{"zerado não sobe sozinho", com(40000, RiskB), 0, nil, 0, false, "não sobe sozinho"},
		{"sem histórico não reduz nem renova", CreditSuggestion{SuggestedLimit: dec(2500), RiskClass: RiskC}, 50000, nil, 50000, false, "revisão manual"},
		{"igual mantém", com(50000, RiskB), 50000, nil, 50000, false, "igual"},
	}
	for _, c := range casos {
		d := DecideAutoReview(c.s, dec(c.atual), c.ultima, pol)
		if !d.NewLimit.Equal(dec(c.quer)) || d.Changed != c.mudou || !strings.Contains(d.Reason, c.contem) {
			t.Errorf("%s: limite %s mudou %v motivo %q", c.nome, d.NewLimit, d.Changed, d.Reason)
		}
		if d.Skip != (c.nome == "sem histórico não reduz nem renova") {
			t.Errorf("%s: skip %v", c.nome, d.Skip)
		}
		if len([]rune(d.Reason)) < 15 {
			t.Errorf("%s: motivo curto demais para a trilha", c.nome)
		}
	}
}

// Cliente sem capital social na Receita (CPF ou CNPJ sem o dado): o faturamento
// declarado mede a capacidade; sem ele, 25% do piso.
func TestCapacidadePeloFaturamentoDeclarado(t *testing.T) {
	hoje := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	pol := DefaultCreditPolicy()
	seller := SellerVolume{Billed12m: dec(12_000_000), Months: 12} // apetite 20% × 1 mi = 200 mil
	semNada := SuggestCreditLimit(CreditHistory{}, CompanyProfile{}, pol, seller, hoje)
	decl := &FinancialStatement{AnnualRevenue: dec(6_000_000), ReferenceDate: hoje.AddDate(0, -6, 0), Source: "DRE"}
	comDecl := SuggestCreditLimit(CreditHistory{}, CompanyProfile{Declared: decl}, pol, seller, hoje)
	// 10% de 500 mil/mês = 50 mil de capacidade; classe C (fator 0,6) = 30 mil.
	if !comDecl.SuggestedLimit.Equal(dec(30000)) {
		t.Fatalf("com DRE de 6 mi: %s, quer 30.000", comDecl.SuggestedLimit)
	}
	if !comDecl.SuggestedLimit.GreaterThan(semNada.SuggestedLimit) {
		t.Fatalf("declaração não aumentou o limite: %s vs %s", comDecl.SuggestedLimit, semNada.SuggestedLimit)
	}
	// Capital baixo (Gerdau tinha capital que não refletia o porte): vale o maior.
	capital := dec(50_000)
	ambos := SuggestCreditLimit(CreditHistory{}, CompanyProfile{ShareCapital: &capital, Declared: decl}, pol, seller, hoje)
	if !ambos.SuggestedLimit.Equal(comDecl.SuggestedLimit) {
		t.Fatalf("capital baixo derrubou a capacidade declarada: %s", ambos.SuggestedLimit)
	}
	vencida := &FinancialStatement{AnnualRevenue: dec(6_000_000), ReferenceDate: hoje.AddDate(-2, 0, 0), Source: "DRE"}
	velha := SuggestCreditLimit(CreditHistory{}, CompanyProfile{Declared: vencida}, pol, seller, hoje)
	if !velha.SuggestedLimit.Equal(semNada.SuggestedLimit) {
		t.Fatalf("declaração de 2 anos ainda contou: %s", velha.SuggestedLimit)
	}
}

// Empresa recém-implantada: com menos de 3 meses de notas, o apetite usa o
// faturamento mensal informado na política.
func TestApetiteNaImplantacao(t *testing.T) {
	pol := DefaultCreditPolicy()
	pol.DeclaredMonthlyRevenue = dec(2_000_000)
	ap, como := apetiteDaEmpresa(pol, SellerVolume{Billed12m: dec(100_000), Months: 1})
	if !ap.Equal(dec(400_000)) || !strings.Contains(como, "informado na política") {
		t.Fatalf("apetite %s (%s), quer 400.000 pelo informado", ap, como)
	}
	ap, _ = apetiteDaEmpresa(pol, SellerVolume{Billed12m: dec(3_000_000), Months: 6})
	if !ap.Equal(dec(100_000)) {
		t.Fatalf("com 6 meses de notas vale o sistema: %s", ap)
	}
}
