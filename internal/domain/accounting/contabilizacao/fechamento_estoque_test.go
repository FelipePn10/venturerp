package contabilizacao

import (
	"testing"

	"github.com/shopspring/decimal"
)

func ptr(v int64) *int64 { return &v }

func soma(ls []Lancamento, conta int64) decimal.Decimal {
	s := decimal.Zero
	for _, l := range ls {
		if l.DebitoID == conta {
			s = s.Add(l.Valor)
		}
		if l.CreditoID == conta {
			s = s.Sub(l.Valor)
		}
	}
	return s
}

var contas = ContasProducao{Estoque: ptr(1), Processo: ptr(2), Absorvido: ptr(3), Variacao: ptr(4)}

// Material 100, mão de obra 30, entregue por 130: o processo zera e o estoque
// fica 30 maior, a contrapartida da mão de obra.
func TestProducaoOrdemCompletaZeraOProcesso(t *testing.T) {
	a := ApurarOrdem(OrdemNoMes{Consumo: d("100"), Entrega: d("130"), Encerrada: true})
	if !a.Absorvido.Equal(d("30")) || !a.Variacao.IsZero() {
		t.Fatalf("absorvido %s variação %s", a.Absorvido, a.Variacao)
	}
	ls, falta := Producao(a, contas, "11/2026")
	if len(falta) > 0 {
		t.Fatal(falta)
	}
	if !soma(ls, 2).IsZero() || !soma(ls, 1).Equal(d("30")) || !soma(ls, 3).Equal(d("-30")) {
		t.Fatalf("processo %s estoque %s absorvido %s", soma(ls, 2), soma(ls, 1), soma(ls, 3))
	}
}

// Consumo num mês e entrega no outro: o material fica em processo na virada.
func TestProducaoAtravessaOMes(t *testing.T) {
	m1 := ApurarOrdem(OrdemNoMes{Consumo: d("100")})
	l1, _ := Producao(m1, contas, "10/2026")
	if !soma(l1, 2).Equal(d("100")) {
		t.Fatalf("processo no mês 1: %s", soma(l1, 2))
	}
	m2 := ApurarOrdem(OrdemNoMes{Entrega: d("130"), ConsumoAntes: d("100"), Encerrada: true})
	l2, _ := Producao(m2, contas, "11/2026")
	if !soma(l1, 2).Add(soma(l2, 2)).IsZero() {
		t.Fatalf("processo não zerou: %s", soma(l1, 2).Add(soma(l2, 2)))
	}
}

// Entrega parcial acima do consumo e depois mais material: o absorvido recua.
func TestProducaoAbsorvidoAcompanhaOAcumulado(t *testing.T) {
	a := ApurarOrdem(OrdemNoMes{Consumo: d("60"), ConsumoAntes: d("50"), EntregaAntes: d("80"), AbsorvidoAntes: d("30"), Entrega: d("50"), Encerrada: true})
	if !a.Absorvido.Equal(d("-10")) {
		t.Fatalf("absorvido %s, quero -10", a.Absorvido)
	}
	if !a.Variacao.IsZero() {
		t.Fatalf("variação %s", a.Variacao)
	}
}

// Ordem de beneficiamento consome e nunca entrega: encerrada, o material vai ao CMV.
func TestProducaoOrdemEncerradaSemEntregaVaiAoResultado(t *testing.T) {
	a := ApurarOrdem(OrdemNoMes{Consumo: d("40"), Encerrada: true})
	if !a.Variacao.Equal(d("40")) {
		t.Fatalf("variação %s", a.Variacao)
	}
	ls, _ := Producao(a, contas, "11/2026")
	if !soma(ls, 2).IsZero() || !soma(ls, 4).Equal(d("40")) {
		t.Fatalf("processo %s cmv %s", soma(ls, 2), soma(ls, 4))
	}
}

// Devolução maior que o consumo do mês inverte a partida, sem valor negativo.
func TestProducaoConsumoNegativoInverte(t *testing.T) {
	ls, _ := Producao(ApuracaoOrdem{Consumo: d("-5")}, contas, "11/2026")
	if len(ls) != 1 || ls[0].DebitoID != 1 || ls[0].CreditoID != 2 || !ls[0].Valor.Equal(d("5")) {
		t.Fatalf("%+v", ls)
	}
}

func TestProducaoApontaAContaQueFalta(t *testing.T) {
	_, falta := Producao(ApuracaoOrdem{Consumo: d("5")}, ContasProducao{Estoque: ptr(1)}, "11/2026")
	if len(falta) != 1 {
		t.Fatalf("%v", falta)
	}
	if _, falta := Producao(ApuracaoOrdem{Consumo: d("5")}, contas, "11/2026"); len(falta) != 0 {
		t.Fatalf("%v", falta)
	}
}

func TestProvisaoLancaSoAVariacao(t *testing.T) {
	ls, _ := ProvisaoObsoletos(d("150"), d("100"), ptr(8), ptr(9), "11/2026")
	if len(ls) != 1 || ls[0].DebitoID != 8 || !ls[0].Valor.Equal(d("50")) {
		t.Fatalf("%+v", ls)
	}
	ls, _ = ProvisaoObsoletos(d("80"), d("100"), ptr(8), ptr(9), "11/2026")
	if len(ls) != 1 || ls[0].DebitoID != 9 || !ls[0].Valor.Equal(d("20")) {
		t.Fatalf("reversão: %+v", ls)
	}
	if ls, falta := ProvisaoObsoletos(d("100"), d("100"), nil, nil, ""); ls != nil || falta != nil {
		t.Fatal("sem variação não lança nem cobra conta")
	}
}
