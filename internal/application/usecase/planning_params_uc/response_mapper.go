package planning_params_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
)

func toPlanningParamResponse(p *entity.PlanningParam) *response.PlanningParamResponse {
	if p == nil {
		return nil
	}
	return &response.PlanningParamResponse{
		ID:          p.ID,
		ParamNumber: p.ParamNumber,
		ParamKey:    p.ParamKey,
		Value:       p.Value,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		UpdatedBy:   p.UpdatedBy,
	}
}

func toPlanningParamResponses(list []*entity.PlanningParam) []*response.PlanningParamResponse {
	out := make([]*response.PlanningParamResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toPlanningParamResponse(p))
	}
	return out
}
