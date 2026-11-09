package item_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

type catalogoFake map[string]bool

func (c catalogoFake) UnidadesForaDoCatalogo(_ context.Context, codigos ...string) ([]string, error) {
	var fora []string
	for _, x := range codigos {
		if !c[x] {
			fora = append(fora, x)
		}
	}
	return fora, nil
}

func itemComUnidades(estoque, compra string) *entity.Item {
	i := &entity.Item{}
	i.Warehouse.UnitOfMeasurement = types.TypeUnitOfMeasurementItem(estoque)
	if compra != "" {
		u := types.TypeUnitOfMeasurementItem(compra)
		i.Supplies.PurchaseUOM = &u
	}
	return i
}

func TestUnidadesDoItemNoCatalogo(t *testing.T) {
	cat := catalogoFake{"UN": true, "KG": true}
	if err := validarUnidadesNoCatalogo(context.Background(), cat, itemComUnidades("UN", "KG"), nil); err != nil {
		t.Fatalf("unidades do catálogo recusadas: %v", err)
	}
	err := validarUnidadesNoCatalogo(context.Background(), cat, itemComUnidades("UN", "KGS"), nil)
	if err == nil || !strings.Contains(err.Error(), "KGS (unidade de compra)") {
		t.Fatalf("esperava recusa de KGS na unidade de compra, veio %v", err)
	}
	// Edição: unidade legada que não mudou não bloqueia.
	legado := itemComUnidades("UN", "KGS")
	if err := validarUnidadesNoCatalogo(context.Background(), cat, legado, unidadesDoItem(legado)); err != nil {
		t.Fatalf("unidade legada inalterada bloqueou a edição: %v", err)
	}
	// Sem catálogo injetado, não confere.
	if err := validarUnidadesNoCatalogo(context.Background(), nil, legado, nil); err != nil {
		t.Fatal(err)
	}
}
