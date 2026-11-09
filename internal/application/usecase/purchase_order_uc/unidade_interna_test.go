package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	uomrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

type unidadeFixa map[int64]string

func (u unidadeFixa) UnidadeDeEstoqueDoItem(_ context.Context, item int64) (string, error) {
	return u[item], nil
}

type conversorFixo struct {
	ports.UOMConverter
	fator map[string]float64 // "ITEM:DE>PARA"
}

func (c conversorFixo) ConvertQuantity(_ context.Context, item int64, q decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	f, ok := c.fator[fmt.Sprint(item)+":"+de+">"+para]
	return q.Mul(decimal.NewFromFloat(f)), ok, nil
}

func ptrS(s string) *string { return &s }

func TestUnidadeInterna(t *testing.T) {
	ctx := context.Background()
	u := &UnidadeInterna{
		Unidades:  unidadeFixa{238: "M", 239: "M", 300: "UN"},
		Conversor: conversorFixo{fator: map[string]float64{"238:KG>M": 0.15530965}},
	}

	// Compra em KG de item estocado em M: converte quantidade e preço.
	l := &entity.PurchaseOrderItem{ItemCode: 238, RequestedQty: decimal.NewFromFloat(115.8975), UnitPrice: decimal.NewFromFloat(6.246), PurchaseUOM: ptrS("kg"), InternalUOM: ptrS("KG")}
	if err := u.Preencher(ctx, l); err != nil {
		t.Fatal(err)
	}
	if *l.InternalUOM != "M" || l.InternalQty.LessThan(decimal.NewFromFloat(17.9999)) || l.InternalQty.GreaterThan(decimal.NewFromFloat(18.0001)) {
		t.Fatalf("esperava 18 M (unidade do cadastro, não a informada), veio %v %s", l.InternalQty, *l.InternalUOM)
	}
	if valor := l.InternalPrice.Mul(l.InternalQty).InexactFloat64(); valor < 723.88 || valor > 723.90 {
		t.Fatalf("o valor da linha tem de ser o mesmo nas duas unidades: %v", valor)
	}

	// Sem conversão cadastrada: recusa (antes gravava 19,4 KG como 19,4 M).
	l = &entity.PurchaseOrderItem{ItemCode: 239, RequestedQty: decimal.NewFromFloat(19.4371), UnitPrice: decimal.NewFromFloat(7), PurchaseUOM: ptrS("KG")}
	if err := u.Preencher(ctx, l); err == nil || !strings.Contains(err.Error(), "cadastre a conversão KG→M") {
		t.Fatalf("esperava recusa por falta de conversão, veio %v", err)
	}

	// Sem unidade de compra (sugestão do MRP): está na unidade de estoque.
	l = &entity.PurchaseOrderItem{ItemCode: 300, RequestedQty: decimal.NewFromFloat(10), UnitPrice: decimal.NewFromFloat(2)}
	if err := u.Preencher(ctx, l); err != nil || !l.InternalQty.Equal(decimal.NewFromFloat(10)) || *l.PurchaseUOM != "UN" || *l.InternalUOM != "UN" {
		t.Fatalf("linha sem unidade fica na de estoque: %+v %v", l, err)
	}

	// Sem provedor (testes antigos): mantém o comportamento anterior.
	var nada *UnidadeInterna
	l = &entity.PurchaseOrderItem{ItemCode: 1, RequestedQty: decimal.NewFromFloat(3), UnitPrice: decimal.NewFromFloat(4)}
	if err := nada.Preencher(ctx, l); err != nil || !l.InternalQty.Equal(decimal.NewFromFloat(3)) || !l.InternalPrice.Equal(decimal.NewFromFloat(4)) {
		t.Fatalf("sem provedor: %+v %v", l, err)
	}
}

type catalogoFixo map[string]string // texto → código do catálogo

func (c catalogoFixo) Resolver(_ context.Context, _ *int64, textos []string) (map[string]uomrepo.Resolucao, error) {
	out := map[string]uomrepo.Resolucao{}
	for _, t := range textos {
		out[t] = uomrepo.Resolucao{Texto: t, Unidade: c[t]}
	}
	return out, nil
}

// Sinônimo do catálogo (BMB = BOMBONA) é a própria unidade de estoque: não
// pede conversão BMB→BOMBONA. Unidade que o catálogo não conhece é recusada
// como fora do catálogo, não como "falta conversão".
func TestUnidadeInternaTraduzSinonimoDoCatalogo(t *testing.T) {
	ctx := context.Background()
	u := &UnidadeInterna{
		Unidades:  unidadeFixa{9010: "BOMBONA"},
		Conversor: conversorFixo{fator: map[string]float64{"9010:L>BOMBONA": 0.02}},
		Catalogo:  catalogoFixo{"BMB": "BOMBONA", "L": "L", "LT": "L"},
	}
	l := &entity.PurchaseOrderItem{ItemCode: 9010, RequestedQty: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(200), PurchaseUOM: ptrS("bmb")}
	if err := u.Preencher(ctx, l); err != nil || *l.PurchaseUOM != "BOMBONA" || !l.InternalQty.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("BMB é BOMBONA: esperava 2 BOMBONA sem conversão, veio %v %v %v", err, *l.PurchaseUOM, l.InternalQty)
	}
	l = &entity.PurchaseOrderItem{ItemCode: 9010, RequestedQty: decimal.NewFromInt(100), UnitPrice: decimal.NewFromInt(4), PurchaseUOM: ptrS("LT")}
	if err := u.Preencher(ctx, l); err != nil || *l.PurchaseUOM != "L" || !l.InternalQty.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("100 LT = 100 L = 2 BOMBONA, veio %v %v %v", err, *l.PurchaseUOM, l.InternalQty)
	}
	l = &entity.PurchaseOrderItem{ItemCode: 9010, RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1), PurchaseUOM: ptrS("BOMBONAS")}
	if err := u.Preencher(ctx, l); err == nil || !strings.Contains(err.Error(), "fora do catálogo") {
		t.Fatalf("BOMBONAS não existe no catálogo, veio %v", err)
	}
}
