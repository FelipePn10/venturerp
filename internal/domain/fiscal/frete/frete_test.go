package frete

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func p(v int64) *int64           { return &v }

func itens() []Item {
	return []Item{
		{ItemID: 1, ItemCode: p(10), WarehouseID: p(1), Descricao: "chapa", ValorContabil: d("50000"), Quantidade: d("10000"), EmEstoque: d("10000"), MovimentaEstoque: true},
		{ItemID: 2, ItemCode: p(11), WarehouseID: p(1), Descricao: "luva", ValorContabil: d("50000"), Quantidade: d("100"), EmEstoque: d("40"), MovimentaEstoque: true},
	}
}

func soma(ps []Parte) decimal.Decimal {
	t := decimal.Zero
	for _, x := range ps {
		t = t.Add(x.Valor)
		if !x.ValorEstoque.Add(x.ValorDespesa).Equal(x.Valor) {
			panic("estoque + despesa ≠ parte")
		}
	}
	return t
}

// Frete de 1.200 com 144 de ICMS creditado: 1.056 de custo, metade para cada
// item pelo valor; a luva já consumiu 60% — essa parte vai para despesa.
func TestRatearPorValorSeparaEstoqueEDespesa(t *testing.T) {
	custo := CustoDoFrete(d("1200"), d("144"), true)
	if !custo.Equal(d("1056")) {
		t.Fatalf("custo %s", custo)
	}
	ps, err := Ratear(custo, "valor", itens())
	if err != nil {
		t.Fatal(err)
	}
	if !soma(ps).Equal(custo) || !ps[0].Valor.Equal(d("528")) || !ps[0].ValorEstoque.Equal(d("528")) {
		t.Fatalf("chapa: %+v", ps[0])
	}
	if !ps[1].ValorEstoque.Equal(d("211.2")) || !ps[1].ValorDespesa.Equal(d("316.8")) {
		t.Fatalf("luva: estoque %s despesa %s", ps[1].ValorEstoque, ps[1].ValorDespesa)
	}
}

func TestRatearPorQuantidadeFechaOsCentavos(t *testing.T) {
	ps, err := Ratear(d("100"), RateioQuantidade, []Item{
		{Descricao: "a", Quantidade: d("1")}, {Descricao: "b", Quantidade: d("1")}, {Descricao: "c", Quantidade: d("1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !soma(ps).Equal(d("100")) || !ps[2].Valor.Equal(d("33.34")) {
		t.Fatalf("partes: %v %v %v", ps[0].Valor, ps[1].Valor, ps[2].Valor)
	}
	// Item que não movimenta estoque: tudo despesa.
	if !ps[0].ValorDespesa.Equal(ps[0].Valor) {
		t.Fatal("sem estoque, o frete é despesa")
	}
}

func TestRatearPorPeso(t *testing.T) {
	kg := func(s string) *decimal.Decimal { v := d(s); return &v }
	is := itens()
	is[0].PesoUnitario, is[1].PesoUnitario = kg("1"), kg("0.5") // 10.000 kg × 50 kg
	ps, err := Ratear(d("1005"), RateioPeso, is)                // 10.050 kg no total
	if err != nil {
		t.Fatal(err)
	}
	if !ps[0].Valor.Equal(d("1000")) || !ps[1].Valor.Equal(d("5")) {
		t.Fatalf("peso: %s %s", ps[0].Valor, ps[1].Valor)
	}
	is[1].PesoUnitario = nil
	if _, err := Ratear(d("10"), RateioPeso, is); err == nil {
		t.Fatal("item sem peso deveria impedir o rateio por peso")
	}
}

func TestRatearRecusaSemItensOuTipoInvalido(t *testing.T) {
	if _, err := Ratear(d("10"), RateioValor, nil); err == nil {
		t.Fatal("sem itens")
	}
	if _, err := Ratear(d("10"), "VOLUME", itens()); err == nil {
		t.Fatal("tipo inválido")
	}
	if _, err := Ratear(d("0"), RateioValor, itens()); err == nil {
		t.Fatal("custo zero")
	}
}

func TestPesoEmKg(t *testing.T) {
	if !PesoEmKg(d("1500"), "g").Equal(d("1.5")) || !PesoEmKg(d("2"), "T").Equal(d("2000")) || !PesoEmKg(d("3"), "KG").Equal(d("3")) {
		t.Fatal("conversão de peso")
	}
}
