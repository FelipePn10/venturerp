package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
)

type unidadeFixa map[int64]string

func (u unidadeFixa) UnidadeDeEstoqueDoItem(_ context.Context, item int64) (string, error) {
	return u[item], nil
}

type conversorFixo struct {
	ports.UOMConverter
	fator map[string]float64 // "ITEM:DE>PARA"
}

func (c conversorFixo) ConvertQuantity(_ context.Context, item int64, q float64, de, para string) (float64, bool, error) {
	f, ok := c.fator[fmt.Sprint(item)+":"+de+">"+para]
	return q * f, ok, nil
}

func ptrS(s string) *string { return &s }

func TestUnidadeInterna(t *testing.T) {
	ctx := context.Background()
	u := &UnidadeInterna{
		Unidades:  unidadeFixa{238: "M", 239: "M", 300: "UN"},
		Conversor: conversorFixo{fator: map[string]float64{"238:KG>M": 0.15530965}},
	}

	// Compra em KG de item estocado em M: converte quantidade e preço.
	l := &entity.PurchaseOrderItem{ItemCode: 238, RequestedQty: 115.8975, UnitPrice: 6.246, PurchaseUOM: ptrS("kg"), InternalUOM: ptrS("KG")}
	if err := u.Preencher(ctx, l); err != nil {
		t.Fatal(err)
	}
	if *l.InternalUOM != "M" || l.InternalQty < 17.9999 || l.InternalQty > 18.0001 {
		t.Fatalf("esperava 18 M (unidade do cadastro, não a informada), veio %v %s", l.InternalQty, *l.InternalUOM)
	}
	if valor := l.InternalPrice * l.InternalQty; valor < 723.88 || valor > 723.90 {
		t.Fatalf("o valor da linha tem de ser o mesmo nas duas unidades: %v", valor)
	}

	// Sem conversão cadastrada: recusa (antes gravava 19,4 KG como 19,4 M).
	l = &entity.PurchaseOrderItem{ItemCode: 239, RequestedQty: 19.4371, UnitPrice: 7, PurchaseUOM: ptrS("KG")}
	if err := u.Preencher(ctx, l); err == nil || !strings.Contains(err.Error(), "cadastre a conversão KG→M") {
		t.Fatalf("esperava recusa por falta de conversão, veio %v", err)
	}

	// Sem unidade de compra (sugestão do MRP): está na unidade de estoque.
	l = &entity.PurchaseOrderItem{ItemCode: 300, RequestedQty: 10, UnitPrice: 2}
	if err := u.Preencher(ctx, l); err != nil || l.InternalQty != 10 || *l.PurchaseUOM != "UN" || *l.InternalUOM != "UN" {
		t.Fatalf("linha sem unidade fica na de estoque: %+v %v", l, err)
	}

	// Sem provedor (testes antigos): mantém o comportamento anterior.
	var nada *UnidadeInterna
	l = &entity.PurchaseOrderItem{ItemCode: 1, RequestedQty: 3, UnitPrice: 4}
	if err := nada.Preencher(ctx, l); err != nil || l.InternalQty != 3 || l.InternalPrice != 4 {
		t.Fatalf("sem provedor: %+v %v", l, err)
	}
}
