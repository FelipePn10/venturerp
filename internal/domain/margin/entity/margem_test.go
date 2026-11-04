package entity

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func perto[T float64 | decimal.Decimal](a T, b, tol float64) bool {
	switch v := any(a).(type) {
	case decimal.Decimal:
		return math.Abs(v.InexactFloat64()-b) <= tol
	case float64:
		return math.Abs(v-b) <= tol
	}
	return false
}

// TestTaxaFinanceiraRealConfereComOExemploDoFocco valida a fórmula contra o
// exemplo publicado no help do FoccoERP (FCST0108): ciclo de caixa de 25 dias
// com taxa de 3% ao mês resulta em 2,49%.
func TestTaxaFinanceiraRealConfereComOExemploDoFocco(t *testing.T) {
	p := Parametros{
		FinancialRateMonthly: decimal.NewFromFloat(3),
		AvgSalesTermDays:     25, // ciclo de caixa = 25 - 0 + 0
	}
	if got := p.CicloDeCaixaDias(); got != 25 {
		t.Fatalf("ciclo de caixa = %d, esperado 25", got)
	}
	got := p.TaxaFinanceiraRealPct()
	if !perto(got, 2.49, 0.01) {
		t.Fatalf("taxa financeira real = %v%%, o help do FoccoERP documenta 2,49%%", got)
	}
}

// TestCicloDeCaixaSomaProducaoESubtraiCompra: prazo de venda − prazo de compra
// + ciclo de produção, como o FCST0108 define.
func TestCicloDeCaixaSomaProducaoESubtraiCompra(t *testing.T) {
	p := Parametros{AvgSalesTermDays: 45, AvgPurchaseTermDays: 30, ProductionCycleDays: 10}
	if got := p.CicloDeCaixaDias(); got != 25 {
		t.Fatalf("ciclo = %d, esperado 45-30+10 = 25", got)
	}
	// Recebe antes de pagar: não vira despesa financeira.
	neg := Parametros{AvgSalesTermDays: 0, AvgPurchaseTermDays: 60, ProductionCycleDays: 5}
	if got := neg.CicloDeCaixaDias(); got != 0 {
		t.Fatalf("ciclo negativo = %d, esperado 0", got)
	}
}

// TestCascataDaMargem confere a estrutura inteira com números redondos.
func TestCascataDaMargem(t *testing.T) {
	v := Venda{
		Quantidade: decimal.NewFromFloat(10), ValorLiquido: decimal.NewFromFloat(100), // faturamento mercadoria = 1000
		IPI: decimal.NewFromFloat(50), ICMS: decimal.NewFromFloat(180), PIS: decimal.NewFromFloat(16.5), COFINS: decimal.NewFromFloat(76),
		CustoMateriaPrima: decimal.NewFromFloat(400), CustoTransformacao: decimal.NewFromFloat(150),
		ComissaoValor: decimal.NewFromFloat(30),
	}
	p := Parametros{IRPct: decimal.NewFromFloat(34), AdminPct: decimal.NewFromFloat(5), FreightPct: decimal.NewFromFloat(2)}

	r := Calcular(v, p)

	if !perto(r.FaturamentoBruto, 1050, 0.01) {
		t.Fatalf("faturamento bruto = %v, esperado 1050 (10×100 + 50 de IPI)", r.FaturamentoBruto)
	}
	if !perto(r.FaturamentoMercadoria, 1000, 0.01) {
		t.Fatalf("faturamento da mercadoria = %v, esperado 1000", r.FaturamentoMercadoria)
	}
	// 1000 − 180 − 92,50 − 400 − 150 = 177,50
	if !perto(r.LucroBruto, 177.50, 0.01) {
		t.Fatalf("lucro bruto = %v, esperado 177,50", r.LucroBruto)
	}
	if !perto(r.DespesaAdministrativa, 50, 0.01) || !perto(r.Frete, 20, 0.01) {
		t.Fatalf("adm = %v (esperado 50), frete = %v (esperado 20)", r.DespesaAdministrativa, r.Frete)
	}
	// 177,50 − 50 − 30 − 20 = 77,50 antes do IR
	if !perto(r.ProvisaoIR, 26.35, 0.01) {
		t.Fatalf("provisão de IR = %v, esperado 26,35 (34%% de 77,50)", r.ProvisaoIR)
	}
	if !perto(r.Margem, 51.15, 0.01) {
		t.Fatalf("margem = %v, esperado 51,15", r.Margem)
	}
	if !perto(r.MargemPct, 5.115, 0.01) {
		t.Fatalf("margem %% = %v, esperado 5,115%% sobre o faturamento da mercadoria", r.MargemPct)
	}
}

// TestPrejuizoNaoGeraProvisaoDeIR: não se provisiona imposto sobre prejuízo.
func TestPrejuizoNaoGeraProvisaoDeIR(t *testing.T) {
	v := Venda{Quantidade: decimal.NewFromFloat(1), ValorLiquido: decimal.NewFromFloat(100), CustoMateriaPrima: decimal.NewFromFloat(300)}
	r := Calcular(v, Parametros{IRPct: decimal.NewFromFloat(34)})
	if !r.ProvisaoIR.IsZero() {
		t.Fatalf("provisão de IR = %v sobre prejuízo, esperado 0", r.ProvisaoIR)
	}
	if !r.Margem.IsNegative() {
		t.Fatalf("margem = %v, esperada negativa", r.Margem)
	}
}

// TestDespesaFinanceiraVemDoDescasamento: receber depois de pagar custa; o
// contrário não vira receita na margem.
func TestDespesaFinanceiraVemDoDescasamento(t *testing.T) {
	v := Venda{Quantidade: decimal.NewFromFloat(1), ValorLiquido: decimal.NewFromFloat(1000), CustoMateriaPrima: decimal.NewFromFloat(500)}

	// Recebe em 60 dias, paga a matéria-prima à vista: há custo financeiro.
	comDescasamento := Calcular(v, Parametros{FinancialRateMonthly: decimal.NewFromFloat(2), AvgSalesTermDays: 60})
	if !comDescasamento.DespesaFinanceira.IsPositive() {
		t.Fatalf("despesa financeira = %v, esperada positiva", comDescasamento.DespesaFinanceira)
	}

	// Recebe à vista e paga em 60: favorável, mas não vira receita.
	favoravel := Calcular(v, Parametros{FinancialRateMonthly: decimal.NewFromFloat(2), MaterialPaymentDays: 60})
	if !favoravel.DespesaFinanceira.IsZero() {
		t.Fatalf("despesa financeira = %v, esperado 0 quando o caixa é favorável", favoravel.DespesaFinanceira)
	}

	// Sem taxa não há despesa financeira, qualquer que seja o prazo.
	semTaxa := Calcular(v, Parametros{AvgSalesTermDays: 90})
	if !semTaxa.DespesaFinanceira.IsZero() {
		t.Fatalf("sem taxa a despesa financeira deve ser 0, veio %v", semTaxa.DespesaFinanceira)
	}
}
