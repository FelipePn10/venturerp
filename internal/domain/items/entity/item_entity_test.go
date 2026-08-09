package entity

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

func TestItemValidateCommercialAccounting(t *testing.T) {
	item := validItemForValidation()
	saleType := "VENDA"
	cest := "0100100"
	origin := 0
	factor := decimal.NewFromInt(1)
	item.Commercial.SaleType = &saleType
	item.Commercial.VolumeConversionFactor = &factor
	item.Accounting.CEST = &cest
	item.Accounting.Origin = &origin
	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}
	badCEST := "ABC"
	item.Accounting.CEST = &badCEST
	if err := item.Validate(); err == nil {
		t.Fatal("expected malformed CEST error")
	}
	item.Accounting.CEST = &cest
	badType := "BONIFICACAO"
	item.Commercial.SaleType = &badType
	if err := item.Validate(); err == nil {
		t.Fatal("expected invalid sale type error")
	}
	item.Commercial.SaleType = &saleType
	self := int64(item.Code)
	item.Commercial.PackagingItemCode = &self
	if err := item.Validate(); err == nil {
		t.Fatal("expected packaging self-reference error")
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

func TestItemValidateAcceptsOptionalItemBaseForEveryNature(t *testing.T) {
	tests := []struct {
		name   string
		nature ItemNature
		base   *int
	}{
		{name: "generic without base", nature: ItemGeneric},
		{name: "configured without base", nature: ItemConfigured},
		{name: "item base without base", nature: ItemBase},
		{name: "generic with base", nature: ItemGeneric, base: intPointer(99)},
		{name: "configured with base", nature: ItemConfigured, base: intPointer(99)},
		{name: "item base with base", nature: ItemBase, base: intPointer(99)},
		{name: "zero is treated as absent", nature: ItemConfigured, base: intPointer(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := validItemForValidation()
			item.Nature = tt.nature
			item.Engineering.ItemBaseCod = tt.base
			if err := item.Validate(); err != nil {
				t.Fatal(err)
			}
			if tt.base != nil && *tt.base == 0 && item.Engineering.ItemBaseCod != nil {
				t.Fatal("expected zero item-base code to be normalized as absent")
			}
		})
	}
}

func TestItemValidateTrimsAndRequiresNameInPortuguese(t *testing.T) {
	item := validItemForValidation()
	item.Name = "  Transformador 30 kVA  "
	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}
	if item.Name != "Transformador 30 kVA" {
		t.Fatalf("expected trimmed name, got %q", item.Name)
	}

	item.Name = "   "
	if err := item.Validate(); err == nil || err.Error() != "Informe o nome do item." {
		t.Fatalf("expected Portuguese name validation error, got %v", err)
	}
}

func intPointer(value int) *int { return &value }
