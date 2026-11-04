// Package entity calcula a margem de contribuição — quanto sobra de cada venda
// depois de tudo o que ela consome.
//
// A estrutura segue o FoccoERP (FCST0254 gera, FCST0320 apresenta):
//
//	Faturamento bruto        (valor líquido × qtde + IPI)
//	− IPI                  = Faturamento da mercadoria
//	− ICMS
//	− PIS + COFINS
//	− Custo de matéria-prima
//	− Custo de transformação (mão de obra + gastos gerais)
//	                       = Lucro bruto
//	− Despesa administrativa
//	− Comissão
//	− Frete
//	− Outros
//	− Despesa financeira
//	− Provisão de IR/CSLL
//	                       = Margem  (e % sobre o faturamento da mercadoria)
//
// O que diferencia esta apuração de um "preço menos custo" é a despesa
// financeira: a venda é recebida em 45 dias mas a matéria-prima foi paga em 30,
// e esse descasamento tem custo. É por isso que um pedido com margem contábil
// boa pode dar prejuízo no caixa.
package entity

import (
	"github.com/shopspring/decimal"
)

// Parametros são os percentuais e prazos do mês, cadastrados uma vez e usados
// por todas as vendas do período (FoccoERP FCST0108).
type Parametros struct {
	Ano int
	Mes int

	// Percentuais aplicados sobre o faturamento da mercadoria.
	IRPct                decimal.Decimal // provisão de IR/CSLL
	AdminPct             decimal.Decimal // incidência administrativa rateada entre os itens
	FreightPct           decimal.Decimal // frete médio
	FinancialRateMonthly decimal.Decimal // taxa financeira ao mês, em %

	// Ciclo de caixa: quanto tempo o dinheiro fica fora do bolso.
	AvgSalesTermDays    int // prazo médio de venda
	AvgPurchaseTermDays int // prazo médio de compra
	ProductionCycleDays int // ciclo de produção

	// Prazos de pagamento de cada componente, para trazer a valor presente.
	MaterialPaymentDays int
	LaborPaymentDays    int
	IPIPaymentDays      int
	ICMSPaymentDays     int
	PISPaymentDays      int
	COFINSPaymentDays   int
}

// CicloDeCaixaDias é o intervalo entre pagar e receber:
// prazo médio de venda − prazo médio de compra + ciclo de produção.
//
// Ciclo negativo (recebe antes de pagar) é financeiramente favorável e não vira
// despesa; tratamos como zero para não gerar "receita financeira" fictícia numa
// apuração de margem.
func (p Parametros) CicloDeCaixaDias() int {
	ciclo := p.AvgSalesTermDays - p.AvgPurchaseTermDays + p.ProductionCycleDays
	if ciclo < 0 {
		return 0
	}
	return ciclo
}

// TaxaFinanceiraRealPct ajusta a taxa mensal ao ciclo de caixa, capitalizando:
//
//	(1 + taxa)^(ciclo/30) − 1
//
// O help do FoccoERP traz o exemplo que confirma a fórmula: ciclo de 25 dias e
// taxa de 3% ao mês resultam em 2,49% — que é (1,03)^(25/30) − 1.
func (p Parametros) TaxaFinanceiraRealPct() decimal.Decimal {
	if !p.FinancialRateMonthly.IsPositive() {
		return decimal.Zero
	}
	ciclo := p.CicloDeCaixaDias()
	if ciclo <= 0 {
		return decimal.Zero
	}
	fator := fatorDoPrazo(p.FinancialRateMonthly, ciclo)
	return fator.Sub(um).Mul(cem)
}

var (
	um  = decimal.NewFromInt(1)
	cem = decimal.NewFromInt(100)
	// precisaoPotencia: casas da potência de expoente fracionário — muito além
	// das 4 casas que a margem grava.
	precisaoPotencia int32 = 16
)

// fatorDoPrazo é (1 + taxa/100)^(dias/30), a capitalização da taxa mensal no prazo.
func fatorDoPrazo(taxaMensalPct decimal.Decimal, dias int) decimal.Decimal {
	base := um.Add(taxaMensalPct.Div(cem))
	exp := decimal.NewFromInt(int64(dias)).Div(decimal.NewFromInt(30))
	f, err := base.PowWithPrecision(exp, precisaoPotencia)
	if err != nil {
		// Base sempre > 1 aqui (taxa positiva): o erro não acontece; sem fator,
		// o valor fica como está, que é o comportamento sem taxa.
		return um
	}
	return f
}

// Venda são os valores de uma linha de nota fiscal (ou de pedido) já apurados.
type Venda struct {
	Quantidade   decimal.Decimal
	ValorLiquido decimal.Decimal // preço unitário líquido
	IPI          decimal.Decimal
	ICMS         decimal.Decimal
	PIS          decimal.Decimal
	COFINS       decimal.Decimal

	// Custos vindos da valorização: matéria-prima e transformação
	// (mão de obra + gastos gerais de fabricação), já para a quantidade vendida.
	CustoMateriaPrima  decimal.Decimal
	CustoTransformacao decimal.Decimal

	ComissaoValor decimal.Decimal
	OutrosValor   decimal.Decimal
}

// Resultado é a cascata completa, em decimal. A resposta HTTP é montada no
// caso de uso (margin_uc.ResultadoJSON), com números.
type Resultado struct {
	FaturamentoBruto      decimal.Decimal
	IPI                   decimal.Decimal
	FaturamentoMercadoria decimal.Decimal
	ICMS                  decimal.Decimal
	PISCOFINS             decimal.Decimal
	CustoMateriaPrima     decimal.Decimal
	CustoTransformacao    decimal.Decimal
	LucroBruto            decimal.Decimal
	DespesaAdministrativa decimal.Decimal
	Comissao              decimal.Decimal
	Frete                 decimal.Decimal
	Outros                decimal.Decimal
	DespesaFinanceira     decimal.Decimal
	ProvisaoIR            decimal.Decimal
	Margem                decimal.Decimal
	MargemPct             decimal.Decimal
}

// valorPresente traz um valor a receber/pagar em `dias` para hoje, à taxa
// mensal informada. Sem taxa ou sem prazo, o valor é o próprio.
func valorPresente(valor decimal.Decimal, dias int, taxaMensalPct decimal.Decimal) decimal.Decimal {
	if valor.IsZero() || dias <= 0 || !taxaMensalPct.IsPositive() {
		return valor
	}
	return valor.Div(fatorDoPrazo(taxaMensalPct, dias))
}

// Calcular monta a cascata da margem para uma linha de venda.
//
// A despesa financeira é o descasamento de caixa: o quanto a empresa perde por
// receber depois e pagar antes. Cada parcela é trazida a valor presente pelo
// seu próprio prazo — receber R$ 100 em 60 dias não vale R$ 100 hoje, e pagar
// o ICMS em 30 dias alivia parte disso.
func Calcular(v Venda, p Parametros) Resultado {
	r := Resultado{
		IPI:                v.IPI,
		ICMS:               v.ICMS,
		PISCOFINS:          v.PIS.Add(v.COFINS),
		CustoMateriaPrima:  v.CustoMateriaPrima,
		CustoTransformacao: v.CustoTransformacao,
		Comissao:           v.ComissaoValor,
		Outros:             v.OutrosValor,
	}

	r.FaturamentoBruto = v.ValorLiquido.Mul(v.Quantidade).Add(v.IPI)
	r.FaturamentoMercadoria = r.FaturamentoBruto.Sub(v.IPI)

	r.LucroBruto = r.FaturamentoMercadoria.Sub(r.ICMS).Sub(r.PISCOFINS).
		Sub(r.CustoMateriaPrima).Sub(r.CustoTransformacao)

	base := r.FaturamentoMercadoria
	r.DespesaAdministrativa = base.Mul(p.AdminPct).Div(cem)
	r.Frete = base.Mul(p.FreightPct).Div(cem)

	// Descasamento de caixa: receita recebida depois, custos e impostos pagos
	// nos seus prazos.
	taxa := p.FinancialRateMonthly
	perdaNaReceita := r.FaturamentoBruto.Sub(valorPresente(r.FaturamentoBruto, p.AvgSalesTermDays, taxa))
	ganhoNosPagamentos := decimal.Zero
	for _, parcela := range []struct {
		valor decimal.Decimal
		dias  int
	}{
		{r.CustoMateriaPrima, p.MaterialPaymentDays},
		{r.CustoTransformacao, p.LaborPaymentDays},
		{v.IPI, p.IPIPaymentDays},
		{v.ICMS, p.ICMSPaymentDays},
		{v.PIS, p.PISPaymentDays},
		{v.COFINS, p.COFINSPaymentDays},
	} {
		ganhoNosPagamentos = ganhoNosPagamentos.Add(parcela.valor.Sub(valorPresente(parcela.valor, parcela.dias, taxa)))
	}
	r.DespesaFinanceira = perdaNaReceita.Sub(ganhoNosPagamentos)
	if r.DespesaFinanceira.IsNegative() {
		// Recebe antes de pagar: não vira receita financeira na margem.
		r.DespesaFinanceira = decimal.Zero
	}

	antesDoIR := r.LucroBruto.Sub(r.DespesaAdministrativa).Sub(r.Comissao).
		Sub(r.Frete).Sub(r.Outros).Sub(r.DespesaFinanceira)

	// IR incide sobre lucro; prejuízo não gera provisão.
	if antesDoIR.IsPositive() {
		r.ProvisaoIR = antesDoIR.Mul(p.IRPct).Div(cem)
	}
	r.Margem = antesDoIR.Sub(r.ProvisaoIR)

	if !r.FaturamentoMercadoria.IsZero() {
		r.MargemPct = r.Margem.Div(r.FaturamentoMercadoria).Mul(cem)
	}
	return r
}
