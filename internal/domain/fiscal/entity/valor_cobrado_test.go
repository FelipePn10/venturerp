package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestRetornoDeIndustrializacao(t *testing.T) {
	for cfop, quer := range map[string]bool{
		"5902": true, "5.902": true, "6902": true, "5903": true, "6.903": true,
		"5124": false, "5101": false, "1902": false, "5901": false, "": false, "59020": false,
	} {
		if got := RetornoDeIndustrializacao(cfop); got != quer {
			t.Errorf("CFOP %q: esperado %v, veio %v", cfop, quer, got)
		}
	}
}

// A NF-e 5.911 real: total R$ 412,60, fatura R$ 230,00.
func TestValorCobradoDaNF5911(t *testing.T) {
	nota := &FiscalExit{ValorTotal: 412.60}
	itens := []*FiscalExitItem{
		{Cfop: "5124", TotalPrice: 230.00},
		{Cfop: "5902", TotalPrice: 142.11},
		{Cfop: "5902", TotalPrice: 40.49},
	}
	if got := ValorCobrado(nota, itens); !got.Equal(decimal.RequireFromString("230")) {
		t.Fatalf("valor cobrado = %s, esperado 230,00", got)
	}
}

func TestValorCobradoSemRetornoEOTotal(t *testing.T) {
	nota := &FiscalExit{ValorTotal: 1500.75}
	if got := ValorCobrado(nota, []*FiscalExitItem{{Cfop: "5101", TotalPrice: 1500.75}}); !got.Equal(decimal.RequireFromString("1500.75")) {
		t.Fatalf("venda comum deve cobrar o total, veio %s", got)
	}
	if got := ValorCobrado(nota, nil); !got.Equal(decimal.RequireFromString("1500.75")) {
		t.Fatalf("sem linhas carregadas, vale o total: %s", got)
	}
}

func TestValorCobradoSoDeRetornoEZero(t *testing.T) {
	nota := &FiscalExit{ValorTotal: 182.60}
	itens := []*FiscalExitItem{{Cfop: "5902", TotalPrice: 182.60}}
	if got := ValorCobrado(nota, itens); !got.IsZero() {
		t.Fatalf("nota só de retorno não cobra nada, veio %s", got)
	}
}
