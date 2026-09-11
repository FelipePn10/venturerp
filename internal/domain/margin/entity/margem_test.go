package entity

import (
	"math"
	"testing"
)

func perto(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// TestTaxaFinanceiraRealConfereComOExemploDoFocco valida a fórmula contra o
// exemplo publicado no help do FoccoERP (FCST0108): ciclo de caixa de 25 dias
// com taxa de 3% ao mês resulta em 2,49%.
func TestTaxaFinanceiraRealConfereComOExemploDoFocco(t *testing.T) {
	p := Parametros{
		FinancialRateMonthly: 3,
		AvgSalesTermDays:     25, // ciclo de caixa = 25 - 0 + 0
	}
	if got := p.CicloDeCaixaDias(); got != 25 {
		t.Fatalf("ciclo de caixa = %d, esperado 25", got)
	}
	got := p.TaxaFinanceiraRealPct()
	if !perto(got, 2.49, 0.01) {
		t.Fatalf("taxa financeira real = %.4f%%, o help do FoccoERP documenta 2,49%%", got)
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
		Quantidade: 10, ValorLiquido: 100, // faturamento mercadoria = 1000
		IPI: 50, ICMS: 180, PIS: 16.5, COFINS: 76,
		CustoMateriaPrima: 400, CustoTransformacao: 150,
		ComissaoValor: 30,
	}
	p := Parametros{IRPct: 34, AdminPct: 5, FreightPct: 2}

	r := Calcular(v, p)

	if !perto(r.FaturamentoBruto, 1050, 0.01) {
		t.Fatalf("faturamento bruto = %.2f, esperado 1050 (10×100 + 50 de IPI)", r.FaturamentoBruto)
	}
	if !perto(r.FaturamentoMercadoria, 1000, 0.01) {
		t.Fatalf("faturamento da mercadoria = %.2f, esperado 1000", r.FaturamentoMercadoria)
	}
	// 1000 − 180 − 92,50 − 400 − 150 = 177,50
	if !perto(r.LucroBruto, 177.50, 0.01) {
		t.Fatalf("lucro bruto = %.2f, esperado 177,50", r.LucroBruto)
	}
	if !perto(r.DespesaAdministrativa, 50, 0.01) || !perto(r.Frete, 20, 0.01) {
		t.Fatalf("adm = %.2f (esperado 50), frete = %.2f (esperado 20)", r.DespesaAdministrativa, r.Frete)
	}
	// 177,50 − 50 − 30 − 20 = 77,50 antes do IR
	if !perto(r.ProvisaoIR, 26.35, 0.01) {
		t.Fatalf("provisão de IR = %.2f, esperado 26,35 (34%% de 77,50)", r.ProvisaoIR)
	}
	if !perto(r.Margem, 51.15, 0.01) {
		t.Fatalf("margem = %.2f, esperado 51,15", r.Margem)
	}
	if !perto(r.MargemPct, 5.115, 0.01) {
		t.Fatalf("margem %% = %.3f, esperado 5,115%% sobre o faturamento da mercadoria", r.MargemPct)
	}
}

// TestPrejuizoNaoGeraProvisaoDeIR: não se provisiona imposto sobre prejuízo.
func TestPrejuizoNaoGeraProvisaoDeIR(t *testing.T) {
	v := Venda{Quantidade: 1, ValorLiquido: 100, CustoMateriaPrima: 300}
	r := Calcular(v, Parametros{IRPct: 34})
	if r.ProvisaoIR != 0 {
		t.Fatalf("provisão de IR = %.2f sobre prejuízo, esperado 0", r.ProvisaoIR)
	}
	if r.Margem >= 0 {
		t.Fatalf("margem = %.2f, esperada negativa", r.Margem)
	}
}

// TestDespesaFinanceiraVemDoDescasamento: receber depois de pagar custa; o
// contrário não vira receita na margem.
func TestDespesaFinanceiraVemDoDescasamento(t *testing.T) {
	v := Venda{Quantidade: 1, ValorLiquido: 1000, CustoMateriaPrima: 500}

	// Recebe em 60 dias, paga a matéria-prima à vista: há custo financeiro.
	comDescasamento := Calcular(v, Parametros{FinancialRateMonthly: 2, AvgSalesTermDays: 60})
	if comDescasamento.DespesaFinanceira <= 0 {
		t.Fatalf("despesa financeira = %.2f, esperada positiva", comDescasamento.DespesaFinanceira)
	}

	// Recebe à vista e paga em 60: favorável, mas não vira receita.
	favoravel := Calcular(v, Parametros{FinancialRateMonthly: 2, MaterialPaymentDays: 60})
	if favoravel.DespesaFinanceira != 0 {
		t.Fatalf("despesa financeira = %.2f, esperado 0 quando o caixa é favorável", favoravel.DespesaFinanceira)
	}

	// Sem taxa não há despesa financeira, qualquer que seja o prazo.
	semTaxa := Calcular(v, Parametros{AvgSalesTermDays: 90})
	if semTaxa.DespesaFinanceira != 0 {
		t.Fatalf("sem taxa a despesa financeira deve ser 0, veio %.2f", semTaxa.DespesaFinanceira)
	}
}
