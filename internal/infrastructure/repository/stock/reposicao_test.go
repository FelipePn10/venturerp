package stock

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestQuantidadeDeReposicaoRespeitaMultiploFracionado(t *testing.T) {
	d := decimal.RequireFromString
	casos := []struct {
		nome                                     string
		ponto, seguranca, projetado, min, multip string
		esperado                                 string
	}{
		// Barra de 6,35 M: faltam 10 M, sobem para 2 barras — 12,7 M, não 13.
		{"multiplo fracionado", "10", "0", "0", "0", "6.35", "12.7"},
		{"multiplo inteiro", "10", "2", "1.5", "0", "5", "15"},
		{"lote minimo fracionado sem multiplo sobe ao inteiro", "1", "0", "0", "2.5", "0", "3"},
		{"lote minimo e multiplo fracionados", "1", "0", "0", "7", "6.35", "12.7"},
		{"projecao acima do ponto ainda sugere o minimo de uma unidade", "5", "0", "9", "0", "0", "1"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := quantidadeDeReposicao(d(c.ponto), d(c.seguranca), d(c.projetado), d(c.min), d(c.multip))
			if !got.Equal(d(c.esperado)) {
				t.Fatalf("esperado %s, veio %s", c.esperado, got)
			}
		})
	}
}

func TestPontoDePedidoUsaEstoqueMinimoFracionado(t *testing.T) {
	ponto, motivo := calcularPontoDePedido(nil, decimal.RequireFromString("12.5"), decimal.Zero)
	if !ponto.Equal(decimal.RequireFromString("12.5")) || motivo != "estoque mínimo do cadastro" {
		t.Fatalf("ponto %s motivo %q", ponto, motivo)
	}
	if ponto, _ := calcularPontoDePedido(nil, decimal.Zero, decimal.Zero); !ponto.IsZero() {
		t.Fatalf("sem mínimo nem fórmula o ponto deveria ser zero, veio %s", ponto)
	}
}
