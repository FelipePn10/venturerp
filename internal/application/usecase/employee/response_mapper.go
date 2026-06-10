package employee

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
)

func toEmployeeResponse(e *entity.Employee) *response.EmployeeResponse {
	if e == nil {
		return nil
	}
	return &response.EmployeeResponse{
		ID:                 e.ID,
		Code:               e.Code,
		Name:               e.Name,
		Situation:          string(e.Situation),
		ParticipatesBudget: e.ParticipatesBudget,
		TechnicalAssistant: e.TechnicalAssistant,
		Role:               e.Role,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
		CreatedBy:          e.CreatedBy,
	}
}

func toEmployeeResponses(list []*entity.Employee) []*response.EmployeeResponse {
	out := make([]*response.EmployeeResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toEmployeeResponse(e))
	}
	return out
}
