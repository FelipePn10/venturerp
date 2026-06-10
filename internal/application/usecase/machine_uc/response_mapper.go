package machine_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

func toMachineTypeResponse(t *entity.MachineType) *response.MachineTypeResponse {
	if t == nil {
		return nil
	}
	return &response.MachineTypeResponse{
		Code:             t.Code,
		Name:             t.Name,
		Description:      t.Description,
		Type:             string(t.Type),
		RequiresOperator: t.RequiresOperator,
		IsActive:         t.IsActive,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
		CreatedBy:        t.CreatedBy,
	}
}

func toMachineTypeResponses(list []*entity.MachineType) []*response.MachineTypeResponse {
	out := make([]*response.MachineTypeResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toMachineTypeResponse(t))
	}
	return out
}

func toMachineResponse(m *entity.Machine) *response.MachineResponse {
	if m == nil {
		return nil
	}
	return &response.MachineResponse{
		Code:            m.Code,
		Name:            m.Name,
		MachineTypeCode: m.MachineTypeCode,
		CostCenterCode:  m.CostCenterCode,
		Capacity:        m.Capacity,
		CapacityUnit:    string(m.CapacityUnit),
		CapacityPeriod:  string(m.CapacityPeriod),
		EfficiencyRate:  m.EfficiencyRate,
		IsActive:        m.IsActive,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		CreatedBy:       m.CreatedBy,
	}
}

func toMachineResponses(list []*entity.Machine) []*response.MachineResponse {
	out := make([]*response.MachineResponse, 0, len(list))
	for _, m := range list {
		out = append(out, toMachineResponse(m))
	}
	return out
}

func toItemMachineTimeResponse(t *entity.ItemMachineTime) *response.ItemMachineTimeResponse {
	if t == nil {
		return nil
	}
	return &response.ItemMachineTimeResponse{
		ItemCode:           t.ItemCode,
		Mask:               t.Mask,
		MachineCode:        t.MachineCode,
		ProductionTime:     t.ProductionTime,
		ProductionTimeUnit: string(t.ProductionTimeUnit),
		ProductionBaseQty:  t.ProductionBaseQty,
		SetupTime:          t.SetupTime,
		Priority:           t.Priority,
		IsActive:           t.IsActive,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
}

func toItemMachineTimeResponses(list []*entity.ItemMachineTime) []*response.ItemMachineTimeResponse {
	out := make([]*response.ItemMachineTimeResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toItemMachineTimeResponse(t))
	}
	return out
}

func toMachineScheduleResponse(s *entity.MachineSchedule) *response.MachineScheduleResponse {
	if s == nil {
		return nil
	}
	return &response.MachineScheduleResponse{
		Code:             s.Code,
		MachineCode:      s.MachineCode,
		OrderCode:        s.OrderCode,
		ScheduleDate:     s.ScheduleDate,
		StartTime:        s.StartTime,
		EndTime:          s.EndTime,
		PlannedQty:       s.PlannedQty,
		ProducedQty:      s.ProducedQty,
		Status:           s.Status,
		Sequence:         s.Sequence,
		PriorityOverride: s.PriorityOverride,
		Notes:            s.Notes,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func toMachineScheduleResponses(list []*entity.MachineSchedule) []*response.MachineScheduleResponse {
	out := make([]*response.MachineScheduleResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toMachineScheduleResponse(s))
	}
	return out
}
