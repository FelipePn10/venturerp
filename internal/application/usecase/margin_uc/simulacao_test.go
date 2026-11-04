package margin_uc

import (
	"regexp"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/margin/entity"
)

func params() entity.Parametros {
	return entity.Parametros{
		Ano: 2026, Mes: 9,
		IRPct: decimal.NewFromFloat(34), AdminPct: decimal.NewFromFloat(5), FreightPct: decimal.NewFromFloat(2), FinancialRateMonthly: decimal.NewFromFloat(3),
		AvgSalesTermDays: 45, AvgPurchaseTermDays: 30, ProductionCycleDays: 10,
	}
}

func entrada() SimulacaoEntrada {
	return SimulacaoEntrada{
		Quantidade: decimal.NewFromFloat(50), PrecoUnitario: decimal.NewFromFloat(180), CustoUnitario: decimal.NewFromFloat(60), CustoTransfUnit: decimal.NewFromFloat(22),
		IPIPct: decimal.NewFromFloat(5), ICMSPct: decimal.NewFromFloat(18), PISCOFINSPct: decimal.NewFromFloat(9.25), ComissaoPct: decimal.NewFromFloat(3),
	}
}

// A simulação tem de usar a MESMA cascata da apuração. Se divergirem, o vendedor
// fecha o pedido por um número e o fechamento do mês mostra outro.
func TestSimulacaoUsaAMesmaCascataDaApuracao(t *testing.T) {
	in, p := entrada(), params()
	daSimulacao := entity.Calcular(vendaDe(in, in.PrecoUnitario), p)
	daApuracao := entity.Calcular(entity.Venda{
		Quantidade: decimal.NewFromFloat(50), ValorLiquido: decimal.NewFromFloat(180),
		IPI:               decimal.NewFromFloat(180 * 50 * 0.05),
		ICMS:              decimal.NewFromFloat(180 * 50 * 0.18),
		PIS:               decimal.NewFromFloat(180 * 50 * 0.0925),
		CustoMateriaPrima: decimal.NewFromFloat(60 * 50), CustoTransformacao: decimal.NewFromFloat(22 * 50),
		ComissaoValor: decimal.NewFromFloat(180 * 50 * 0.03),
	}, p)
	if !daSimulacao.Margem.Equal(daApuracao.Margem) || !daSimulacao.MargemPct.Equal(daApuracao.MargemPct) {
		t.Fatalf("simulação (%v) diverge da apuração (%v)", daSimulacao.Margem, daApuracao.Margem)
	}
}

// O preço mínimo é o número que o vendedor leva para a mesa: abaixo dele a
// margem não fecha, e no valor devolvido ela fecha.
func TestPrecoMinimoAtingeAMargemPedidaENaoMenos(t *testing.T) {
	in, p := entrada(), params()
	in.MargemDesejadaPct = decimal.NewFromFloat(25)

	preco, nota := precoParaMargem(in, p)
	if preco == nil {
		t.Fatalf("não achou preço: %s", nota)
	}
	no := entity.Calcular(vendaDe(in, *preco), p).MargemPct
	if no.LessThan(decimal.NewFromFloat(25)) {
		t.Fatalf("no preço devolvido (%s) a margem é %s%%, abaixo dos 25%% pedidos", preco, no)
	}
	abaixo := entity.Calcular(vendaDe(in, preco.Sub(decimal.New(5, -2))), p).MargemPct
	if abaixo.GreaterThanOrEqual(decimal.NewFromInt(25)) {
		t.Fatalf("cinco centavos abaixo (%s) ainda fecha %s%% — o mínimo está alto demais", preco.Sub(decimal.New(5, -2)), abaixo)
	}
}

// Margem impossível precisa dizer isso, não devolver um número inventado.
func TestMargemInatingivelEExplicada(t *testing.T) {
	in, p := entrada(), params()
	in.ICMSPct, in.PISCOFINSPct, in.ComissaoPct = decimal.NewFromFloat(60), decimal.NewFromFloat(30), decimal.NewFromFloat(20) // 110% do faturamento em tributo e comissão
	in.MargemDesejadaPct = decimal.NewFromFloat(30)

	preco, nota := precoParaMargem(in, p)
	if preco != nil {
		t.Fatalf("devolveu preço %v para uma margem impossível", *preco)
	}
	if nota == "" {
		t.Fatal("não explicou por que não há preço")
	}
}

// Quantidade não muda o PERCENTUAL de margem: tudo escala junto.
func TestPercentualDeMargemIndependeDaQuantidade(t *testing.T) {
	p := params()
	a := entrada()
	b := entrada()
	b.Quantidade = decimal.NewFromFloat(5000)
	ma := entity.Calcular(vendaDe(a, a.PrecoUnitario), p).MargemPct
	mb := entity.Calcular(vendaDe(b, b.PrecoUnitario), p).MargemPct
	if ma.Sub(mb).Abs().GreaterThan(decimal.New(1, -9)) {
		t.Fatalf("margem %% mudou com a quantidade: %v vs %v", ma, mb)
	}
}

// Margem inatingível tem de dizer QUANTO dá. "Não é possível" sozinho manda o
// vendedor adivinhar; o teto é a informação que ele foi buscar.
func TestMargemInatingivelInformaOTeto(t *testing.T) {
	in, p := entrada(), params()
	in.MargemDesejadaPct = decimal.NewFromFloat(40) // acima do teto real (~38,7%)

	preco, nota := precoParaMargem(in, p)
	if preco != nil {
		t.Fatalf("devolveu preço %v para margem acima do teto", *preco)
	}
	if !regexp.MustCompile(`máximo é \d+[.,]\d+%`).MatchString(nota) {
		t.Fatalf("a nota não informa o teto: %q", nota)
	}
	// E o teto informado precisa ser verdade: logo abaixo dele tem de haver preço.
	if p2, _ := precoParaMargem(func() SimulacaoEntrada { i := entrada(); i.MargemDesejadaPct = decimal.NewFromFloat(38); return i }(), p); p2 == nil {
		t.Fatal("38% deveria ser alcançável se o teto é ~38,7%")
	}
}
