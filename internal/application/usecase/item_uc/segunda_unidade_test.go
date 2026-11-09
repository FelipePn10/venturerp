package item_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

// catalogoComConversao responde também o fator cadastrado na VSUP0110.
type catalogoComConversao struct {
	catalogoFake
	fator *decimal.Decimal
}

func (c catalogoComConversao) FatorDaConversaoDoItem(_ context.Context, _ int64, de, para string) (*decimal.Decimal, error) {
	if de != "UN" || para != "KG" {
		return nil, nil
	}
	return c.fator, nil
}

func TestSegundaUnidadeNaoContradizAVSUP0110(t *testing.T) {
	ctx := context.Background()
	vinteQuatro := decimal.NewFromInt(24)
	cat := catalogoComConversao{catalogoFake: catalogoFake{"UN": true, "KG": true}, fator: &vinteQuatro}
	item := func(fator string) *entity.Item { return novoItem("UN", "KG", fator) }

	if err := validarSegundaUnidade(ctx, cat, item("24"), nil, nil); err != nil {
		t.Fatalf("mesmo fator da VSUP0110 deveria passar: %v", err)
	}
	err := validarSegundaUnidade(ctx, cat, item("25"), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "VSUP0110") || !strings.Contains(err.Error(), "24") {
		t.Fatalf("esperava recusa citando o 24 da VSUP0110, veio %v", err)
	}
	// Edição que não mexeu na segunda unidade: legado divergente continua editável.
	antesU, antesF := "KG", decimal.NewFromInt(25)
	if err := validarSegundaUnidade(ctx, cat, item("25"), &antesU, &antesF); err != nil {
		t.Fatalf("segunda unidade inalterada não deveria bloquear: %v", err)
	}
	// Sem conversão na VSUP0110, qualquer fator vale.
	if err := validarSegundaUnidade(ctx, catalogoComConversao{catalogoFake: cat.catalogoFake}, item("25"), nil, nil); err != nil {
		t.Fatal(err)
	}
	// Catálogo sem a interface opcional (fakes antigos): não confere.
	if err := validarSegundaUnidade(ctx, cat.catalogoFake, item("25"), nil, nil); err != nil {
		t.Fatal(err)
	}
}

func novoItem(estoque, segunda, fator string) *entity.Item {
	i := itemComUnidades(estoque, "")
	i.Code = 77
	i.Warehouse.SecondaryUOM = &segunda
	f := decimal.RequireFromString(fator)
	i.Warehouse.SecondaryFactor = &f
	return i
}
