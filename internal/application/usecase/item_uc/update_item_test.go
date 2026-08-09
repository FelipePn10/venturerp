package item_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type updateAuth struct{ ports.AuthService }

func (updateAuth) CanCreateItem(context.Context) bool { return true }

type updateRepo struct {
	missingItemRepository
	item *entity.Item
}

func (r updateRepo) FindItemByCode(context.Context, valueobject.ItemCode) (*entity.Item, error) {
	return r.item, nil
}
func (r updateRepo) UpdateCommercialAccounting(_ context.Context, item *entity.Item) (*entity.Item, error) {
	return item, nil
}

func TestUpdateItemMergesOmittedFolderFields(t *testing.T) {
	description := "Original"
	notes := "keep"
	item := &entity.Item{Code: 1, Name: "Item", Nature: entity.ItemBase, Situation: types.LINHA, Health: types.ACTIVE,
		Warehouse: entity.Warehouse{UnitOfMeasurement: types.UN}, Engineering: entity.Engineering{Weight: valueobject.Weight{Gross: 1, Net: 1, Unit: "KG"}, Type: types.FABRICADO, TypeStruct: types.INDUSTRIAL}, Planning: entity.Planning{TypeMRP: types.NORMAL_MRP}, Supplies: entity.Supplies{TypeOfUse: types.INDUSTRIALIZACAO}, Commercial: entity.Commercial{Description: &description, WarrantyDays: 365, Notes: &notes}}
	newDescription := " Updated "
	descriptionPtr := &newDescription
	warranty := 730
	res, err := NewUpdateItemUseCase(updateRepo{item: item}, updateAuth{}).Execute(context.Background(), 1, request.UpdateItemDTO{Commercial: &request.UpdateCommercialDTO{Description: &descriptionPtr, WarrantyDays: &warranty}})
	if err != nil {
		t.Fatal(err)
	}
	if *res.Commercial.Description != "Updated" || res.Commercial.WarrantyDays != 730 || *res.Commercial.Notes != "keep" {
		t.Fatalf("partial update lost data: %+v", res.Commercial)
	}
}
