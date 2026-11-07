package stock

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

func TestPontoDePedidoUsaEstoqueMinimoFracionado(t *testing.T) {
	ponto, motivo := calcularPontoDePedido(nil, decimal.RequireFromString("12.5"), decimal.Zero, decimal.Zero, decimal.Zero, entity.CalendarioDeTrabalho{}, time.Now())
	if !ponto.Equal(decimal.RequireFromString("12.5")) || motivo != "estoque mínimo do cadastro" {
		t.Fatalf("ponto %s motivo %q", ponto, motivo)
	}
	if ponto, _ := calcularPontoDePedido(nil, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, entity.CalendarioDeTrabalho{}, time.Now()); !ponto.IsZero() {
		t.Fatalf("sem mínimo nem fórmula o ponto deveria ser zero, veio %s", ponto)
	}
}

// A fórmula já soma a segurança: somá-la de novo na quantidade pedia a mais.
func TestPontoDePedidoContemASegurancaUmaVezSo(t *testing.T) {
	d := decimal.RequireFromString
	raw := []byte(`{"tr":10,"cm":"30","cr":30,"es":"5"}`)
	if ponto, _ := calcularPontoDePedido(raw, decimal.Zero, d("99"), decimal.Zero, decimal.Zero, entity.CalendarioDeTrabalho{}, time.Now()); !ponto.Equal(d("15")) {
		t.Fatalf("10 × 30 / 30 + 5 = 15, veio %s", ponto)
	}
	semES := []byte(`{"tr":10,"cm":"30","cr":30}`)
	if ponto, _ := calcularPontoDePedido(semES, decimal.Zero, d("7"), decimal.Zero, decimal.Zero, entity.CalendarioDeTrabalho{}, time.Now()); !ponto.Equal(d("17")) {
		t.Fatalf("sem ES na fórmula vale a segurança do cadastro: 17, veio %s", ponto)
	}
	if ponto, _ := calcularPontoDePedido(nil, d("4"), d("6"), decimal.Zero, decimal.Zero, entity.CalendarioDeTrabalho{}, time.Now()); !ponto.Equal(d("6")) {
		t.Fatalf("a segurança é o piso do mínimo: 6, veio %s", ponto)
	}
}

func TestRepartirExcedenteReservado(t *testing.T) {
	d := decimal.RequireFromString
	m := repartirExcedenteReservado([]linhaDeSaldo{
		{endereco: "A1", quantidade: d("5"), reservado: d("0")},
		{endereco: "A2", quantidade: d("0"), reservado: d("5")},
	})
	if !m["A1"].Equal(d("5")) || !m["A2"].Equal(d("-5")) {
		t.Fatalf("o excedente vai para o livre: %v", m)
	}
	m = repartirExcedenteReservado([]linhaDeSaldo{
		{endereco: "A1", quantidade: d("2"), reservado: d("0")},
		{endereco: "A2", quantidade: d("1"), reservado: d("5")},
	})
	if !m["A1"].Equal(d("2")) || !m["A2"].Equal(d("-2")) {
		t.Fatalf("o que não cabe fica onde está: %v", m)
	}
	if m = repartirExcedenteReservado([]linhaDeSaldo{{endereco: "A1", quantidade: d("5"), reservado: d("5")}}); len(m) != 0 {
		t.Fatalf("sem excedente não muda: %v", m)
	}
}

// Por dia trabalhado: o consumo é por dia em que a empresa trabalha e o prazo
// conta os dias que ela vai trabalhar, pelo calendário.
func TestPontoDePedidoPorDiaTrabalhado(t *testing.T) {
	d := decimal.RequireFromString
	sexta := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	raw := []byte(`{"TR":5,"CM":10,"CR":1,"ES":20,"modo":"DIA_TRABALHADO"}`)
	normal := entity.CalendarioDeTrabalho{}
	if p, _ := calcularPontoDePedido(raw, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, normal, sexta); !p.Equal(d("70")) {
		t.Fatalf("10/dia × 5 dias + 20 = 70, veio %s", p)
	}
	sabado := entity.CalendarioDeTrabalho{Excecoes: map[string]bool{"2026-11-07": true}}
	if p, _ := calcularPontoDePedido(raw, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, sabado, sexta); !p.Equal(d("80")) {
		t.Fatalf("sábado trabalhado no prazo: 10 × 6 + 20 = 80, veio %s", p)
	}
	apurar := []byte(`{"TR":5,"CM":0,"CR":1,"ES":0,"modo":"DIA_TRABALHADO"}`)
	p, motivo := calcularPontoDePedido(apurar, decimal.Zero, d("3"), decimal.Zero, d("4.2"), normal, sexta)
	if !p.Equal(d("24")) || motivo != "ponto de pedido com o consumo por dia trabalhado apurado" {
		t.Fatalf("apurado 4,2/dia × 5 + segurança 3 = 24, veio %s (%s)", p, motivo)
	}
}
