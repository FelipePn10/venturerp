package order_priority_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
)

func toOrderPriorityResponse(p *entity.OrderPriority) *response.OrderPriorityResponse {
	if p == nil {
		return nil
	}
	return &response.OrderPriorityResponse{
		Code:          p.Code,
		IntervalStart: p.IntervalStart,
		IntervalEnd:   p.IntervalEnd,
		Priority:      p.Priority,
		Description:   p.Description,
		IsActive:      p.IsActive,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		CreatedBy:     p.CreatedBy,
	}
}

func toOrderPriorityResponses(list []*entity.OrderPriority) []*response.OrderPriorityResponse {
	out := make([]*response.OrderPriorityResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toOrderPriorityResponse(p))
	}
	return out
}
