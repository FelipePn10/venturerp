package entity

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

func validItemForValidation() *Item {
	return &Item{
		Code:      valueobject.ItemCode(1),
		Name:      "Suporte soldado",
		Nature:    ItemBase,
		Situation: types.LINHA,
		Health:    types.ACTIVE,
		Warehouse: Warehouse{UnitOfMeasurement: types.PC},
		Engineering: Engineering{
			Weight: valueobject.Weight{Gross: 1, Net: 1, Unit: "KG"},
			Type:   types.FABRICADO, TypeStruct: types.INDUSTRIAL,
		},
		Planning:  Planning{TypeMRP: types.NORMAL_MRP},
		Supplies:  Supplies{TypeOfUse: types.INDUSTRIALIZACAO},
		CreatedBy: uuid.New(),
	}
}

func TestItemValidateAcceptsNewMasterData(t *testing.T) {
	item := validItemForValidation()
	abc := "A"
	purchaseUOM := types.CX
	item.Planning.ABCClass = &abc
	item.Planning.MinimumLot = 10
	item.Planning.MultipleLot = 5
	item.Planning.SafetyStock = 2
	item.Supplies.PurchaseUOM = &purchaseUOM
	item.Commercial.WarrantyDays = 365
	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestItemValidateRejectsInvalidEnumsAndMasterData(t *testing.T) {
	tests := []func(*Item){
		func(i *Item) { i.Name = " " },
		func(i *Item) { i.Engineering.Type = types.TypeItem(99) },
		func(i *Item) { i.Planning.MinimumLot = -1 },
		func(i *Item) { invalid := "D"; i.Planning.ABCClass = &invalid },
		func(i *Item) {
			invalid := types.TypeUnitOfMeasurementItem("INVALID")
			i.Supplies.PurchaseUOM = &invalid
		},
	}
	for index, mutate := range tests {
		item := validItemForValidation()
		mutate(item)
		if err := item.Validate(); err == nil {
			t.Fatalf("case %d: expected validation error", index)
		}
	}
}
