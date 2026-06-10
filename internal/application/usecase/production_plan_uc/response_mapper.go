package production_plan_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
)

func toProductionPlanResponse(p *entity.ProductionPlan) *response.ProductionPlanResponse {
	if p == nil {
		return nil
	}
	return &response.ProductionPlanResponse{
		ID:                  p.ID,
		Code:                p.Code,
		Name:                p.Name,
		IndependentDemands:  p.IndependentDemands,
		GroupSameDateOrders: p.GroupSameDateOrders,
		PlanningTypes:       p.PlanningTypes,
		Classification:      p.Classification,
		ClassItemCodes:      p.ClassItemCodes,
		OrderItemCode:       p.OrderItemCode,
		LastCalculatedAt:    p.LastCalculatedAt,
		Parameters:          p.Parameters,
		IsActive:            p.IsActive,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
		CreatedBy:           p.CreatedBy,
	}
}

func toProductionPlanResponses(list []*entity.ProductionPlan) []*response.ProductionPlanResponse {
	out := make([]*response.ProductionPlanResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toProductionPlanResponse(p))
	}
	return out
}
