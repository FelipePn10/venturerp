package structure

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

func ToItemStructureDTO(e *entity.ItemStructure) *response.StructureComponentResponse {
	if e == nil {
		return nil
	}

	return &response.StructureComponentResponse{
		ID:                     e.ID,
		ParentCode:             e.ParentCode,
		ChildCode:              e.ChildCode,
		ParentMask:             e.ParentMask,
		IsGeneric:              e.ParentMask == nil,
		Quantity:               e.Quantity.InexactFloat64(),
		NominalQuantity:        e.Quantity.InexactFloat64(),
		QuantityFormula:        e.QuantityFormula,
		QuantityRounding:       e.QuantityRounding,
		QuantityScale:          e.QuantityScale,
		FormulaVariables:       e.FormulaVariables(),
		EffectiveQuantity:      e.EffectiveQuantity().InexactFloat64(),
		QuantityStockUOM:       e.QuantidadeNaUnidadeDeEstoque().InexactFloat64(),
		ConversionFactor:       e.FatorParaEstoque().InexactFloat64(),
		StockUnitOfMeasurement: e.StockUnitOfMeasurement,
		UnitOfMeasurement:      e.UnitOfMeasurement,
		Health:                 e.Health,
		LossPercentage:         e.LossPercentage.InexactFloat64(),
		LossFormula:            e.LossFormula,
		Sequence:               e.Sequence,
		Notes:                  e.Notes,
		StartDate:              e.StartDate,
		EndDate:                e.EndDate,
		IsCoproduct:            e.IsCoproduct,
		IsFixedQty:             e.IsFixedQty,
		SubstituteGroup:        e.SubstituteGroup,
		SubstitutePriority:     e.SubstitutePriority,
		Inherit:                e.Inherit,
		WarehouseCode:          e.WarehouseCode,
		LineWarehouseCode:      e.LineWarehouseCode,
		SetupLoss:              e.SetupLoss,
		CostLossType:           e.CostLossType,
		CostLoss:               e.CostLoss,
		CostCenterCode:         e.CostCenterCode,
		IsCriticalMPS:          e.IsCriticalMPS,
		GeneratesInspection:    e.GeneratesInspection,
		CustomerSupplied:       e.CustomerSupplied,
		IsActive:               e.IsActive,
		CreatedBy:              e.CreatedBy,
		CreatedAt:              e.CreatedAt,
		UpdatedAt:              e.UpdatedAt,
	}
}

func ToItemStructureListDTO(items []*entity.ItemStructure) []*response.StructureComponentResponse {
	result := make([]*response.StructureComponentResponse, 0, len(items))

	for _, item := range items {
		result = append(result, ToItemStructureDTO(item))
	}

	return result
}
