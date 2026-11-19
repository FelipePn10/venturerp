package item_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type recalculoFake struct {
	chamadas []int64
	n        int
	avisos   []string
}

func (r *recalculoFake) RecalcularDoComponente(_ context.Context, item int64, _ string) (int, []string, error) {
	r.chamadas = append(r.chamadas, item)
	return r.n, r.avisos, nil
}
func (r *recalculoFake) RecalcularDaEmpresa(context.Context, string) (int, []string, error) {
	return 0, nil, nil
}

// Trocar a unidade de estoque do tubo (MM → M) recalcula as estruturas em que
// ele é componente; mudar outra coisa do item não recalcula nada.
func TestTrocarUnidadeDeEstoqueRecalculaAsEstruturas(t *testing.T) {
	novoItem := func() *entity.Item {
		return &entity.Item{Code: 101, Name: "Tubo", Nature: entity.ItemBase, Situation: types.LINHA, Health: types.ACTIVE,
			Warehouse:   entity.Warehouse{UnitOfMeasurement: types.MM},
			Engineering: entity.Engineering{Weight: valueobject.Weight{Gross: 1, Net: 1, Unit: "KG"}, Type: types.COMPRADO, TypeStruct: types.INDUSTRIAL},
			Planning:    entity.Planning{TypeMRP: types.NORMAL_MRP}, Supplies: entity.Supplies{TypeOfUse: types.INDUSTRIALIZACAO}}
	}
	metro := types.M
	rec := &recalculoFake{n: 3, avisos: []string{"linha sem conversão"}}
	uc := NewUpdateItemUseCase(updateRepo{item: novoItem()}, updateAuth{})
	uc.Estruturas = rec
	res, err := uc.Execute(context.Background(), 101, request.UpdateItemDTO{Warehouse: &request.UpdateWarehouseDTO{UnitOfMeasurement: &metro}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.chamadas) != 1 || rec.chamadas[0] != 101 {
		t.Fatalf("recálculo: %v", rec.chamadas)
	}
	if len(res.Warnings) != 2 || !strings.Contains(res.Warnings[0], "3 linha(s)") || !strings.Contains(res.Warnings[0], "M") {
		t.Fatalf("avisos: %v", res.Warnings)
	}

	rec2 := &recalculoFake{}
	uc2 := NewUpdateItemUseCase(updateRepo{item: novoItem()}, updateAuth{})
	uc2.Estruturas = rec2
	nome := "Tubo 1\""
	if _, err := uc2.Execute(context.Background(), 101, request.UpdateItemDTO{Name: &nome}); err != nil {
		t.Fatal(err)
	}
	if len(rec2.chamadas) != 0 {
		t.Fatal("mudou só o nome e recalculou as estruturas")
	}
}
