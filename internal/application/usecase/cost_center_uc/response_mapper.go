package cost_center_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
)

func toCostCenterResponse(c *entity.CostCenter) *response.CostCenterResponse {
	if c == nil {
		return nil
	}
	return &response.CostCenterResponse{
		ID:          c.ID,
		Code:        c.Code,
		Description: c.Description,
		ParentCode:  c.ParentCode,
		Type:        string(c.Type),
		IsRatio:     c.IsRatio,
		StartDate:   c.StartDate,
		EndDate:     c.EndDate,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		CreatedBy:   c.CreatedBy,
	}
}

func toCostCenterResponses(list []*entity.CostCenter) []*response.CostCenterResponse {
	out := make([]*response.CostCenterResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCostCenterResponse(c))
	}
	return out
}
