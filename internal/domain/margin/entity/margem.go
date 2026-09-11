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

import "math"

// Parametros são os percentuais e prazos do mês, cadastrados uma vez e usados
// por todas as vendas do período (FoccoERP FCST0108).
type Parametros struct {
	Ano int
	Mes int

	// Percentuais aplicados sobre o faturamento da mercadoria.
	IRPct                float64 // provisão de IR/CSLL
	AdminPct             float64 // incidência administrativa rateada entre os itens
	FreightPct           float64 // frete médio
	FinancialRateMonthly float64 // taxa financeira ao mês, em %

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
func (p Parametros) TaxaFinanceiraRealPct() float64 {
	if p.FinancialRateMonthly <= 0 {
		return 0
	}
	ciclo := float64(p.CicloDeCaixaDias())
	if ciclo <= 0 {
		return 0
	}
	taxa := p.FinancialRateMonthly / 100.0
	return (math.Pow(1+taxa, ciclo/30.0) - 1) * 100.0
}

// Venda são os valores de uma linha de nota fiscal (ou de pedido) já apurados.
type Venda struct {
	Quantidade   float64
	ValorLiquido float64 // preço unitário líquido
	IPI          float64
	ICMS         float64
	PIS          float64
	COFINS       float64

	// Custos vindos da valorização: matéria-prima e transformação
	// (mão de obra + gastos gerais de fabricação), já para a quantidade vendida.
	CustoMateriaPrima  float64
	CustoTransformacao float64

	ComissaoValor float64
	OutrosValor   float64
}

// Resultado é a cascata completa, pronta para a tela e para o relatório.
type Resultado struct {
	FaturamentoBruto      float64 `json:"faturamento_bruto"`
	IPI                   float64 `json:"ipi"`
	FaturamentoMercadoria float64 `json:"faturamento_mercadoria"`
	ICMS                  float64 `json:"icms"`
	PISCOFINS             float64 `json:"pis_cofins"`
	CustoMateriaPrima     float64 `json:"custo_materia_prima"`
	CustoTransformacao    float64 `json:"custo_transformacao"`
	LucroBruto            float64 `json:"lucro_bruto"`
	DespesaAdministrativa float64 `json:"despesa_administrativa"`
	Comissao              float64 `json:"comissao"`
	Frete                 float64 `json:"frete"`
	Outros                float64 `json:"outros"`
	DespesaFinanceira     float64 `json:"despesa_financeira"`
	ProvisaoIR            float64 `json:"provisao_ir"`
	Margem                float64 `json:"margem"`
	MargemPct             float64 `json:"margem_pct"`
}

// valorPresente traz um valor a receber/pagar em `dias` para hoje, à taxa
// mensal informada. Sem taxa ou sem prazo, o valor é o próprio.
func valorPresente(valor float64, dias int, taxaMensalPct float64) float64 {
	if valor == 0 || dias <= 0 || taxaMensalPct <= 0 {
		return valor
	}
	return valor / math.Pow(1+taxaMensalPct/100.0, float64(dias)/30.0)
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
		PISCOFINS:          v.PIS + v.COFINS,
		CustoMateriaPrima:  v.CustoMateriaPrima,
		CustoTransformacao: v.CustoTransformacao,
		Comissao:           v.ComissaoValor,
		Outros:             v.OutrosValor,
	}

	r.FaturamentoBruto = v.ValorLiquido*v.Quantidade + v.IPI
	r.FaturamentoMercadoria = r.FaturamentoBruto - v.IPI

	r.LucroBruto = r.FaturamentoMercadoria - r.ICMS - r.PISCOFINS -
		r.CustoMateriaPrima - r.CustoTransformacao

	base := r.FaturamentoMercadoria
	r.DespesaAdministrativa = base * p.AdminPct / 100.0
	r.Frete = base * p.FreightPct / 100.0

	// Descasamento de caixa: receita recebida depois, custos e impostos pagos
	// nos seus prazos.
	taxa := p.FinancialRateMonthly
	perdaNaReceita := r.FaturamentoBruto - valorPresente(r.FaturamentoBruto, p.AvgSalesTermDays, taxa)
	ganhoNosPagamentos := 0.0
	for _, parcela := range []struct {
		valor float64
		dias  int
	}{
		{r.CustoMateriaPrima, p.MaterialPaymentDays},
		{r.CustoTransformacao, p.LaborPaymentDays},
		{v.IPI, p.IPIPaymentDays},
		{v.ICMS, p.ICMSPaymentDays},
		{v.PIS, p.PISPaymentDays},
		{v.COFINS, p.COFINSPaymentDays},
	} {
		ganhoNosPagamentos += parcela.valor - valorPresente(parcela.valor, parcela.dias, taxa)
	}
	r.DespesaFinanceira = perdaNaReceita - ganhoNosPagamentos
	if r.DespesaFinanceira < 0 {
		// Recebe antes de pagar: não vira receita financeira na margem.
		r.DespesaFinanceira = 0
	}

	antesDoIR := r.LucroBruto - r.DespesaAdministrativa - r.Comissao -
		r.Frete - r.Outros - r.DespesaFinanceira

	// IR incide sobre lucro; prejuízo não gera provisão.
	if antesDoIR > 0 {
		r.ProvisaoIR = antesDoIR * p.IRPct / 100.0
	}
	r.Margem = antesDoIR - r.ProvisaoIR

	if r.FaturamentoMercadoria != 0 {
		r.MargemPct = r.Margem / r.FaturamentoMercadoria * 100.0
	}
	return r
}
