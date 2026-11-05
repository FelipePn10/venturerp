package stock

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestPontoDePedidoUsaEstoqueMinimoFracionado(t *testing.T) {
	ponto, motivo := calcularPontoDePedido(nil, decimal.RequireFromString("12.5"), decimal.Zero, decimal.Zero)
	if !ponto.Equal(decimal.RequireFromString("12.5")) || motivo != "estoque mínimo do cadastro" {
		t.Fatalf("ponto %s motivo %q", ponto, motivo)
	}
	if ponto, _ := calcularPontoDePedido(nil, decimal.Zero, decimal.Zero, decimal.Zero); !ponto.IsZero() {
		t.Fatalf("sem mínimo nem fórmula o ponto deveria ser zero, veio %s", ponto)
	}
}

// A fórmula já soma a segurança: somá-la de novo na quantidade pedia a mais.
func TestPontoDePedidoContemASegurancaUmaVezSo(t *testing.T) {
	d := decimal.RequireFromString
	raw := []byte(`{"tr":10,"cm":"30","cr":30,"es":"5"}`)
	if ponto, _ := calcularPontoDePedido(raw, decimal.Zero, d("99"), decimal.Zero); !ponto.Equal(d("15")) {
		t.Fatalf("10 × 30 / 30 + 5 = 15, veio %s", ponto)
	}
	semES := []byte(`{"tr":10,"cm":"30","cr":30}`)
	if ponto, _ := calcularPontoDePedido(semES, decimal.Zero, d("7"), decimal.Zero); !ponto.Equal(d("17")) {
		t.Fatalf("sem ES na fórmula vale a segurança do cadastro: 17, veio %s", ponto)
	}
	if ponto, _ := calcularPontoDePedido(nil, d("4"), d("6"), decimal.Zero); !ponto.Equal(d("6")) {
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
