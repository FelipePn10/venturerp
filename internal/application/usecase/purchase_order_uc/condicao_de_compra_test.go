package purchase_order_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
)

type estoqueKG struct{}

func (estoqueKG) UnidadeDeEstoqueDoItem(context.Context, int64) (string, error) { return "KG", nil }

// 1 UN (chapa) = 84,78 KG.
type conversorChapa struct{}

func (conversorChapa) fator(de, para string) (decimal.Decimal, bool) {
	um := decimal.RequireFromString("84.78")
	switch {
	case de == "UN" && para == "KG":
		return um, true
	case de == "KG" && para == "UN":
		return decimal.NewFromInt(1).Div(um), true
	}
	return decimal.Zero, false
}
func (c conversorChapa) Factor(_ context.Context, _ int64, de, para string) (decimal.Decimal, bool, error) {
	f, ok := c.fator(de, para)
	return f, ok, nil
}
func (c conversorChapa) ConvertQuantity(_ context.Context, _ int64, q decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	f, ok := c.fator(de, para)
	return q.Mul(f), ok, nil
}
func (c conversorChapa) ConvertUnitPrice(_ context.Context, _ int64, p decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	f, ok := c.fator(de, para)
	if !ok {
		return decimal.Zero, false, nil
	}
	return p.Div(f), true, nil
}

type condicaoFixa struct{ c CondicaoDeCompra }

func (f condicaoFixa) CondicaoDeCompra(context.Context, int64, int64, string) (CondicaoDeCompra, error) {
	return f.c, nil
}

func TestSugestaoSaiNaUnidadeDeCompraComPrecoDaTabela(t *testing.T) {
	d := decimal.RequireFromString
	novaLinha := func() *poentity.PurchaseOrderItem {
		return &poentity.PurchaseOrderItem{ItemCode: 7, RequestedQty: d("376.7932")}
	}
	uc := &ApprovePurchaseSuggestionUseCase{
		Unidade:   &UnidadeInterna{Unidades: estoqueKG{}, Conversor: conversorChapa{}},
		Condicoes: condicaoFixa{CondicaoDeCompra{Unidade: "UN", Casas: 0, Precos: []PrecoDeTabela{{Preco: d("12.50"), Unidade: "KG", Tabela: 3}}}},
	}

	// 376,7932 kg ÷ 84,78 = 4,44 chapas → 5 UN; preço da tabela por KG vira por UN.
	l := novaLinha()
	if err := uc.aplicarCondicaoDeCompra(context.Background(), l, 1, decimal.Zero); err != nil {
		t.Fatal(err)
	}
	if l.PurchaseUOM == nil || *l.PurchaseUOM != "UN" || !l.RequestedQty.Equal(d("5")) {
		t.Fatalf("linha em %v %s, esperado 5 UN", l.PurchaseUOM, l.RequestedQty)
	}
	if !l.UnitPrice.Equal(d("1059.75")) || !l.TotalPrice.Equal(d("5298.75")) {
		t.Fatalf("preço %s total %s, esperado 1.059,75 por chapa (12,50 × 84,78)", l.UnitPrice, l.TotalPrice)
	}
	// O Preencher depois devolve a quantidade de estoque: 5 chapas = 423,9 kg.
	if err := uc.Unidade.Preencher(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	if !l.InternalQty.Equal(d("423.9")) {
		t.Fatalf("quantidade de estoque %s, esperado 423,9 kg", l.InternalQty)
	}

	// Preço do comprador vale por unidade da linha.
	l = novaLinha()
	if err := uc.aplicarCondicaoDeCompra(context.Background(), l, 1, d("1000")); err != nil {
		t.Fatal(err)
	}
	if !l.UnitPrice.Equal(d("1000")) {
		t.Fatalf("preço informado ignorado: %s", l.UnitPrice)
	}

	// Sem preço nenhum: recusa (o pedido a R$ 0,00 passava pela alçada).
	semPreco := &ApprovePurchaseSuggestionUseCase{Unidade: uc.Unidade, Condicoes: condicaoFixa{CondicaoDeCompra{Unidade: "UN", Casas: 0}}}
	if err := semPreco.aplicarCondicaoDeCompra(context.Background(), novaLinha(), 1, decimal.Zero); err == nil || !strings.Contains(err.Error(), "não tem preço") {
		t.Fatalf("aprovou sem preço: %v", err)
	}

	// Unidade de compra sem conversão: fica na de estoque (como antes).
	semConv := &ApprovePurchaseSuggestionUseCase{Unidade: uc.Unidade, Condicoes: condicaoFixa{CondicaoDeCompra{Unidade: "CX", Casas: 0, Precos: []PrecoDeTabela{{Preco: d("3")}}}}}
	l = novaLinha()
	if err := semConv.aplicarCondicaoDeCompra(context.Background(), l, 1, decimal.Zero); err != nil {
		t.Fatal(err)
	}
	if l.PurchaseUOM != nil || !l.RequestedQty.Equal(d("376.7932")) || !l.UnitPrice.Equal(d("3")) {
		t.Fatalf("sem conversão: %v %s %s", l.PurchaseUOM, l.RequestedQty, l.UnitPrice)
	}
}

// A faixa de quantidade está na unidade da linha da tabela. A chapa estoca em
// KG e a tabela vende em UN: "a partir de 10 UN" custa 900, "a partir de 0"
// custa 1.000. Cinco chapas (376 kg) não alcançam a faixa de 10 — antes a
// comparação era 376 kg ≥ 10 e o pedido saía no preço de quem compra 10.
func TestFaixaDaTabelaNaUnidadeDaTabela(t *testing.T) {
	d := decimal.RequireFromString
	faixas := []PrecoDeTabela{{Preco: d("900"), Unidade: "UN", MinQtd: d("10"), Tabela: 4}, {Preco: d("1000"), Unidade: "UN", Tabela: 4}}
	uc := &ApprovePurchaseSuggestionUseCase{
		Unidade:   &UnidadeInterna{Unidades: estoqueKG{}, Conversor: conversorChapa{}},
		Condicoes: condicaoFixa{CondicaoDeCompra{Unidade: "UN", Casas: 0, Precos: faixas}},
	}
	l := &poentity.PurchaseOrderItem{ItemCode: 7, RequestedQty: d("376.7932")}
	if err := uc.aplicarCondicaoDeCompra(context.Background(), l, 1, decimal.Zero); err != nil {
		t.Fatal(err)
	}
	if !l.RequestedQty.Equal(d("5")) || !l.UnitPrice.Equal(d("1000")) {
		t.Fatalf("5 chapas: %s UN a %s, esperado 1.000 (faixa de 0)", l.RequestedQty, l.UnitPrice)
	}
	// Doze chapas (1.017 kg) alcançam a faixa de 10.
	l = &poentity.PurchaseOrderItem{ItemCode: 7, RequestedQty: d("1017.36")}
	if err := uc.aplicarCondicaoDeCompra(context.Background(), l, 1, decimal.Zero); err != nil {
		t.Fatal(err)
	}
	if !l.RequestedQty.Equal(d("12")) || !l.UnitPrice.Equal(d("900")) {
		t.Fatalf("12 chapas: %s UN a %s, esperado 900 (faixa de 10)", l.RequestedQty, l.UnitPrice)
	}
	// Faixa por KG numa tabela em KG compara em KG: 423,9 kg (5 chapas) < 500 kg.
	porKG := &ApprovePurchaseSuggestionUseCase{Unidade: uc.Unidade, Condicoes: condicaoFixa{CondicaoDeCompra{Unidade: "UN", Casas: 0,
		Precos: []PrecoDeTabela{{Preco: d("11"), Unidade: "KG", MinQtd: d("500")}, {Preco: d("12.50"), Unidade: "KG"}}}}}
	l = &poentity.PurchaseOrderItem{ItemCode: 7, RequestedQty: d("376.7932")}
	if err := porKG.aplicarCondicaoDeCompra(context.Background(), l, 1, decimal.Zero); err != nil {
		t.Fatal(err)
	}
	if !l.UnitPrice.Equal(d("1059.75")) {
		t.Fatalf("faixa em KG: %s, esperado 12,50 × 84,78 = 1.059,75", l.UnitPrice)
	}
	// Só existe a faixa de 10 e o pedido é de 5: sem preço, a aprovação recusa.
	so10 := &ApprovePurchaseSuggestionUseCase{Unidade: uc.Unidade, Condicoes: condicaoFixa{CondicaoDeCompra{Unidade: "UN", Casas: 0, Precos: faixas[:1]}}}
	if err := so10.aplicarCondicaoDeCompra(context.Background(), &poentity.PurchaseOrderItem{ItemCode: 7, RequestedQty: d("376.7932")}, 1, decimal.Zero); err == nil || !strings.Contains(err.Error(), "a tabela 4 só tem preço a partir de 10 UN e o pedido é de 5 UN") {
		t.Fatalf("faixa não alcançada virou preço: %v", err)
	}
}
