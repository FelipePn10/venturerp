package margin_uc

import (
	"math"
	"regexp"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/margin/entity"
)

func params() entity.Parametros {
	return entity.Parametros{
		Ano: 2026, Mes: 9,
		IRPct: 34, AdminPct: 5, FreightPct: 2, FinancialRateMonthly: 3,
		AvgSalesTermDays: 45, AvgPurchaseTermDays: 30, ProductionCycleDays: 10,
	}
}

func entrada() SimulacaoEntrada {
	return SimulacaoEntrada{
		Quantidade: 50, PrecoUnitario: 180, CustoUnitario: 60, CustoTransfUnit: 22,
		IPIPct: 5, ICMSPct: 18, PISCOFINSPct: 9.25, ComissaoPct: 3,
	}
}

// A simulação tem de usar a MESMA cascata da apuração. Se divergirem, o vendedor
// fecha o pedido por um número e o fechamento do mês mostra outro.
func TestSimulacaoUsaAMesmaCascataDaApuracao(t *testing.T) {
	in, p := entrada(), params()
	daSimulacao := entity.Calcular(vendaDe(in, in.PrecoUnitario), p)
	daApuracao := entity.Calcular(entity.Venda{
		Quantidade: 50, ValorLiquido: 180,
		IPI:               180 * 50 * 0.05,
		ICMS:              180 * 50 * 0.18,
		PIS:               180 * 50 * 0.0925,
		CustoMateriaPrima: 60 * 50, CustoTransformacao: 22 * 50,
		ComissaoValor: 180 * 50 * 0.03,
	}, p)
	if daSimulacao.Margem != daApuracao.Margem || daSimulacao.MargemPct != daApuracao.MargemPct {
		t.Fatalf("simulação (%v) diverge da apuração (%v)", daSimulacao.Margem, daApuracao.Margem)
	}
}

// O preço mínimo é o número que o vendedor leva para a mesa: abaixo dele a
// margem não fecha, e no valor devolvido ela fecha.
func TestPrecoMinimoAtingeAMargemPedidaENaoMenos(t *testing.T) {
	in, p := entrada(), params()
	in.MargemDesejadaPct = 25

	preco, nota := precoParaMargem(in, p)
	if preco == nil {
		t.Fatalf("não achou preço: %s", nota)
	}
	no := entity.Calcular(vendaDe(in, *preco), p).MargemPct
	if no < 25 {
		t.Fatalf("no preço devolvido (%.2f) a margem é %.4f%%, abaixo dos 25%% pedidos", *preco, no)
	}
	abaixo := entity.Calcular(vendaDe(in, *preco-0.05), p).MargemPct
	if abaixo >= 25 {
		t.Fatalf("cinco centavos abaixo (%.2f) ainda fecha %.4f%% — o mínimo está alto demais", *preco-0.05, abaixo)
	}
}

// Margem impossível precisa dizer isso, não devolver um número inventado.
func TestMargemInatingivelEExplicada(t *testing.T) {
	in, p := entrada(), params()
	in.ICMSPct, in.PISCOFINSPct, in.ComissaoPct = 60, 30, 20 // 110% do faturamento em tributo e comissão
	in.MargemDesejadaPct = 30

	preco, nota := precoParaMargem(in, p)
	if preco != nil {
		t.Fatalf("devolveu preço %.2f para uma margem impossível", *preco)
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
	b.Quantidade = 5000
	ma := entity.Calcular(vendaDe(a, a.PrecoUnitario), p).MargemPct
	mb := entity.Calcular(vendaDe(b, b.PrecoUnitario), p).MargemPct
	if math.Abs(ma-mb) > 1e-9 {
		t.Fatalf("margem %% mudou com a quantidade: %v vs %v", ma, mb)
	}
}

// Margem inatingível tem de dizer QUANTO dá. "Não é possível" sozinho manda o
// vendedor adivinhar; o teto é a informação que ele foi buscar.
func TestMargemInatingivelInformaOTeto(t *testing.T) {
	in, p := entrada(), params()
	in.MargemDesejadaPct = 40 // acima do teto real (~38,7%)

	preco, nota := precoParaMargem(in, p)
	if preco != nil {
		t.Fatalf("devolveu preço %.2f para margem acima do teto", *preco)
	}
	if !regexp.MustCompile(`máximo é \d+[.,]\d+%`).MatchString(nota) {
		t.Fatalf("a nota não informa o teto: %q", nota)
	}
	// E o teto informado precisa ser verdade: logo abaixo dele tem de haver preço.
	if p2, _ := precoParaMargem(func() SimulacaoEntrada { i := entrada(); i.MargemDesejadaPct = 38; return i }(), p); p2 == nil {
		t.Fatal("38% deveria ser alcançável se o teto é ~38,7%")
	}
}
