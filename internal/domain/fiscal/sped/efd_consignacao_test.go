package sped

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// O que está com o cliente ou é do fornecedor sai no H010 com o dono e o participante.
func TestBlocoHSeparaAPosseDaConsignacao(t *testing.T) {
	cliente := Participante{Cod: "11222333000181", Nome: "Cliente", CNPJ: "11222333000181"}
	item := ItemCadastro{Cod: "1", Desc: "Item", UnidInv: "UN"}
	p := Montar(DadosEFD{DataInventario: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), InventarioItens: []ItemInventario{
		{Item: item, Quantidade: decimal.NewFromInt(4), Valor: decimal.NewFromInt(40)},
		{Item: item, Quantidade: decimal.NewFromInt(6), Valor: decimal.NewFromInt(60), IndProp: "1", Parte: &cliente},
	}})
	if len(p.Inventario) != 2 || p.Inventario[0].IndProp != "0" || p.Inventario[0].CodPart != "" {
		t.Fatalf("%+v", p.Inventario)
	}
	if p.Inventario[1].IndProp != "1" || p.Inventario[1].CodPart != cliente.Cod {
		t.Fatalf("%+v", p.Inventario[1])
	}
	if len(p.Participantes) != 1 {
		t.Fatalf("o participante precisa do 0150: %+v", p.Participantes)
	}
}
