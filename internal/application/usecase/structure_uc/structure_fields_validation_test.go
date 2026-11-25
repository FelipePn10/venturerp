package structure_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemvo "github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/shopspring/decimal"
)

func testStructureItems() structureBusinessCodeItems {
	return structureBusinessCodeItems{items: map[itemvo.BusinessCode]*itementity.Item{
		"PAI-100":   {Code: 100, BusinessCode: "PAI-100"},
		"FILHO-200": {Code: 200, BusinessCode: "FILHO-200"},
	}}
}

func TestCreateAndUpdateValidateAllIndicatorsAndFormulas(t *testing.T) {
	items := testStructureItems()
	inicio := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	fimAnterior := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	formulaInvalida := "2*(COMPRIMENTO"
	perdaFormulaValida := "largura * 0.05"
	notas := "Observação técnica"
	almox := int64(3)
	almoxLinha := int64(4)
	centroCusto := int64(7)

	casosErro := []struct {
		nome   string
		dto    request.CreateStructureComponentDTO
		trecho string
	}{
		{
			nome: "vigência invertida",
			dto: request.CreateStructureComponentDTO{
				ParentCode: "PAI-100", ChildCode: "FILHO-200", Sequence: 10,
				Quantity: decimal.NewFromInt(1), UnitOfMeasurement: "UN", Health: "ATIVO",
				StartDate: &inicio, EndDate: &fimAnterior,
			},
			trecho: "vigência",
		},
		{
			nome: "fórmula de perda com parêntese aberto",
			dto: request.CreateStructureComponentDTO{
				ParentCode: "PAI-100", ChildCode: "FILHO-200", Sequence: 10,
				Quantity: decimal.NewFromInt(1), UnitOfMeasurement: "UN", Health: "ATIVO",
				LossFormula: &formulaInvalida,
			},
			trecho: "fórmula de perda",
		},
		{
			nome: "co-produto junto com quantidade fixa por ordem",
			dto: request.CreateStructureComponentDTO{
				ParentCode: "PAI-100", ChildCode: "FILHO-200", Sequence: 10,
				Quantity: decimal.NewFromInt(1), UnitOfMeasurement: "UN", Health: "ATIVO",
				IsCoproduct: true, IsFixedQty: true,
			},
			trecho: "co-produto",
		},
		{
			nome: "co-produto junto com material fornecido pelo cliente",
			dto: request.CreateStructureComponentDTO{
				ParentCode: "PAI-100", ChildCode: "FILHO-200", Sequence: 10,
				Quantity: decimal.NewFromInt(1), UnitOfMeasurement: "UN", Health: "ATIVO",
				IsCoproduct: true, CustomerSupplied: true,
			},
			trecho: "fornecido pelo cliente",
		},
		{
			nome: "perda de preparação negativa",
			dto: request.CreateStructureComponentDTO{
				ParentCode: "PAI-100", ChildCode: "FILHO-200", Sequence: 10,
				Quantity: decimal.NewFromInt(1), UnitOfMeasurement: "UN", Health: "ATIVO",
				SetupLoss: -0.5,
			},
			trecho: "preparação",
		},
		{
			nome: "perda de custo percentual acima de 100",
			dto: request.CreateStructureComponentDTO{
				ParentCode: "PAI-100", ChildCode: "FILHO-200", Sequence: 10,
				Quantity: decimal.NewFromInt(1), UnitOfMeasurement: "UN", Health: "ATIVO",
				CostLossType: "PERCENTUAL", CostLoss: 120,
			},
			trecho: "custo",
		},
	}

	for _, tc := range casosErro {
		t.Run("Create_"+tc.nome, func(t *testing.T) {
			repo := &structureBusinessCodeRepo{}
			uc := NewCreateStructureComponentUseCase(repo, structureBusinessCodeAuth{}, items)
			_, err := uc.Execute(context.Background(), tc.dto)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.trecho) {
				t.Fatalf("esperava erro contendo %q, obteve: %v", tc.trecho, err)
			}
		})
	}

	// Update com todos os indicadores válidos preenchidos e normalizados
	repo := &structureBusinessCodeRepo{}
	updateUC := NewUpdateStructureComponentUseCase(repo, structureBusinessCodeAuth{}, items)
	fimPosterior := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	atualizado, err := updateUC.Execute(context.Background(), request.UpdateStructureComponentDTO{
		ParentCode:          "PAI-100",
		ChildCode:           "FILHO-200",
		Position:            20,
		Quantity:            decimal.NewFromFloat(2.5),
		UnitOfMeasurement:   "UN",
		Health:              "FANTASMA",
		LossPercentage:      decimal.NewFromFloat(5),
		LossFormula:         &perdaFormulaValida,
		QuantityRounding:    "UP",
		QuantityScale:       3,
		Notes:               &notas,
		StartDate:           &inicio,
		EndDate:             &fimPosterior,
		IsFixedQty:          true,
		Inherit:             true,
		SubstituteGroup:     2,
		SubstitutePriority:  1,
		WarehouseCode:       &almox,
		LineWarehouseCode:   &almoxLinha,
		SetupLoss:           1.25,
		CostLossType:        "QUANTIDADE",
		CostLoss:            150, // permitido acima de 100 quando o tipo é QUANTIDADE
		CostCenterCode:      &centroCusto,
		IsCriticalMPS:       true,
		GeneratesInspection: true,
		CustomerSupplied:    true,
	})
	if err != nil {
		t.Fatalf("atualização completa falhou: %v", err)
	}
	if !atualizado.Inherit || !atualizado.IsFixedQty || !atualizado.IsCriticalMPS || !atualizado.GeneratesInspection || !atualizado.CustomerSupplied {
		t.Fatalf("indicadores booleanos não preservados no Update: %+v", atualizado)
	}
	if atualizado.LossFormula == nil || *atualizado.LossFormula != "LARGURA * 0.05" {
		t.Fatalf("LossFormula não normalizada: %v", atualizado.LossFormula)
	}
	if atualizado.WarehouseCode == nil || *atualizado.WarehouseCode != almox ||
		atualizado.LineWarehouseCode == nil || *atualizado.LineWarehouseCode != almoxLinha ||
		atualizado.CostCenterCode == nil || *atualizado.CostCenterCode != centroCusto {
		t.Fatalf("vínculos de consumo/custo perdidos no Update: %+v", atualizado)
	}
	if atualizado.CostLossType != "QUANTIDADE" || atualizado.CostLoss != 150 || atualizado.SetupLoss != 1.25 {
		t.Fatalf("campos de perda perdidos no Update: %+v", atualizado)
	}
}
