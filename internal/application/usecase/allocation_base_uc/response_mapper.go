package allocation_base_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
)

func toAllocationBaseResponse(ab *entity.AllocationBase) *response.AllocationBaseResponse {
	if ab == nil {
		return nil
	}
	return &response.AllocationBaseResponse{
		Code:        ab.Code,
		Description: ab.Description,
		Period:      ab.Period,
		Observation: ab.Observation,
		Items:       toAllocationBaseItemValues(ab.Items),
		CreatedAt:   ab.CreatedAt,
		UpdatedAt:   ab.UpdatedAt,
		CreatedBy:   ab.CreatedBy,
	}
}

func toAllocationBaseResponses(list []*entity.AllocationBase) []*response.AllocationBaseResponse {
	out := make([]*response.AllocationBaseResponse, 0, len(list))
	for _, ab := range list {
		out = append(out, toAllocationBaseResponse(ab))
	}
	return out
}

func toAllocationBaseItemValues(items []*entity.AllocationBaseItem) []response.AllocationBaseItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.AllocationBaseItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, response.AllocationBaseItemResponse{
			AllocationBaseCode: it.AllocationBaseCode,
			CostCenterCode:     it.CostCenterCode,
			Amount:             it.Amount,
			Percentage:         it.Percentage,
			CreatedAt:          it.CreatedAt,
		})
	}
	return out
}
