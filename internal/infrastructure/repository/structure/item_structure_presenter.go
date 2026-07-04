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
		ID:                 e.ID,
		ParentCode:         e.ParentCode,
		ChildCode:          e.ChildCode,
		ParentMask:         e.ParentMask,
		IsGeneric:          e.ParentMask == nil,
		Quantity:           e.Quantity,
		EffectiveQuantity:  e.Quantity * (1 + e.LossPercentage/100),
		UnitOfMeasurement:  e.UnitOfMeasurement,
		Health:             e.Health,
		LossPercentage:     e.LossPercentage,
		LossFormula:        e.LossFormula,
		Sequence:           e.Sequence,
		Notes:              e.Notes,
		StartDate:          e.StartDate,
		EndDate:            e.EndDate,
		IsCoproduct:        e.IsCoproduct,
		IsFixedQty:         e.IsFixedQty,
		SubstituteGroup:    e.SubstituteGroup,
		SubstitutePriority: e.SubstitutePriority,
		IsActive:           e.IsActive,
		CreatedBy:          e.CreatedBy,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

func ToItemStructureListDTO(items []*entity.ItemStructure) []*response.StructureComponentResponse {
	result := make([]*response.StructureComponentResponse, 0, len(items))

	for _, item := range items {
		result = append(result, ToItemStructureDTO(item))
	}

	return result
}
