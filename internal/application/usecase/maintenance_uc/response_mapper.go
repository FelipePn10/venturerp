package maintenance_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity"
)

func toMaintenancePlanResponse(p *entity.MaintenancePlan) *response.MaintenancePlanResponse {
	if p == nil {
		return nil
	}
	return &response.MaintenancePlanResponse{
		ID:              p.ID,
		Code:            p.Code,
		MachineID:       p.MachineID,
		WorkCenterID:    p.WorkCenterID,
		Description:     p.Description,
		Frequency:       string(p.Frequency),
		FrequencyDays:   p.FrequencyDays,
		EstimatedHours:  p.EstimatedHours,
		LastExecutedAt:  p.LastExecutedAt,
		NextScheduledAt: p.NextScheduledAt,
		IsActive:        p.IsActive,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
		CreatedBy:       p.CreatedBy,
	}
}

func toMaintenancePlanResponses(list []*entity.MaintenancePlan) []*response.MaintenancePlanResponse {
	out := make([]*response.MaintenancePlanResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toMaintenancePlanResponse(p))
	}
	return out
}

func toMaintenanceOrderResponse(o *entity.MaintenanceOrder) *response.MaintenanceOrderResponse {
	if o == nil {
		return nil
	}
	return &response.MaintenanceOrderResponse{
		ID:             o.ID,
		PlanID:         o.PlanID,
		MachineID:      o.MachineID,
		WorkCenterID:   o.WorkCenterID,
		ScheduledDate:  o.ScheduledDate,
		EstimatedHours: o.EstimatedHours,
		ActualHours:    o.ActualHours,
		Status:         string(o.Status),
		StartedAt:      o.StartedAt,
		CompletedAt:    o.CompletedAt,
		Notes:          o.Notes,
		IsActive:       o.IsActive,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
}

func toMaintenanceOrderResponses(list []*entity.MaintenanceOrder) []*response.MaintenanceOrderResponse {
	out := make([]*response.MaintenanceOrderResponse, 0, len(list))
	for _, o := range list {
		out = append(out, toMaintenanceOrderResponse(o))
	}
	return out
}
