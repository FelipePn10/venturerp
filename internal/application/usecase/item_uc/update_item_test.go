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
func (r updateRepo) UpdateFolders(_ context.Context, item *entity.Item) (*entity.Item, error) {
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

// A alteração do item só gravava Comercial e Contábil: nome, natureza, situação
// e as pastas de engenharia/planejamento/suprimentos ficavam congeladas depois
// do cadastro. Este teste tranca o comportamento novo — e a preservação do que
// não foi enviado.
func TestUpdateItemAlteraIdentificacaoMarcadoresEPastas(t *testing.T) {
	notes := "manter"
	item := &entity.Item{
		Code: 1, Name: "Nome antigo", Nature: entity.ItemBase, IsBase: true,
		Situation: types.LINHA, Health: types.ACTIVE,
		PDM:         entity.PDM{GroupCode: 1, ModifierCode: 3, DescriptionTechnique: "Descrição antiga"},
		Warehouse:   entity.Warehouse{UnitOfMeasurement: types.UN},
		Engineering: entity.Engineering{Weight: valueobject.Weight{Gross: 1, Net: 1, Unit: "KG"}, Type: types.FABRICADO, TypeStruct: types.INDUSTRIAL},
		Planning:    entity.Planning{TypeMRP: types.NORMAL_MRP, LLC: 1},
		Supplies:    entity.Supplies{TypeOfUse: types.INDUSTRIALIZACAO},
		Commercial:  entity.Commercial{Notes: &notes},
	}

	nome := " Porta de armário 450 "
	configurado := true
	llc := 5
	fantasma := true
	descricaoTecnica := "Porta MDF 450x1800"

	res, err := NewUpdateItemUseCase(updateRepo{item: item}, updateAuth{}).Execute(
		context.Background(), 1,
		request.UpdateItemDTO{
			Name:         &nome,
			IsConfigured: &configurado,
			PDM:          &request.UpdatePDMDTO{DescriptionTechnique: &descricaoTecnica},
			Planning:     &request.UpdatePlanningDTO{LLC: &llc, Ghost: &fantasma},
		})
	if err != nil {
		t.Fatal(err)
	}

	if item.Name != "Porta de armário 450" {
		t.Fatalf("o nome deveria ter sido alterado e aparado, veio %q", item.Name)
	}
	if !item.IsBase || !item.IsConfigured {
		t.Fatalf("item base + configurado deveriam coexistir, veio base=%v configurado=%v", item.IsBase, item.IsConfigured)
	}
	if item.PDM.DescriptionTechnique != "Porta MDF 450x1800" {
		t.Fatalf("descrição técnica não alterada: %q", item.PDM.DescriptionTechnique)
	}
	if item.Planning.LLC != 5 || !item.Planning.Ghost {
		t.Fatalf("planejamento não alterado: llc=%d fantasma=%v", item.Planning.LLC, item.Planning.Ghost)
	}
	// O que não foi enviado continua como estava.
	if item.Situation != types.LINHA || item.Engineering.Type != types.FABRICADO || item.Supplies.TypeOfUse != types.INDUSTRIALIZACAO {
		t.Fatal("campos omitidos deveriam ter sido preservados")
	}
	if res == nil {
		t.Fatal("a alteração deveria devolver o item")
	}
}
