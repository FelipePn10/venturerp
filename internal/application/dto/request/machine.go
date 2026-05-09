package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type CreateMachineTypeDTO struct {
	Code        int64                 `json:"code"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	Type        types.MachineTypeEnum `json:"type"`
	CreatedBy   uuid.UUID             `json:"created_by"`
	IsActive    bool                  `json:"is_active"`
}

type UpdateMachineTypeDTO struct {
	Code        int64   `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Type        string  `json:"type"`
	IsActive    bool    `json:"is_active"`
}

type CreateMachineDTO struct {
	Code            int64                     `json:"code"`
	Name            string                    `json:"name"`
	MachineTypeCode int64                     `json:"machine_type_code"`
	CostCenterCode  *int64                    `json:"cost_center_code,omitempty"`
	Capacity        float64                   `json:"capacity"`
	CapacityUnit    types.MachineCapacityUnit `json:"capacity_per_unit"`
	CapacityPeriod  types.CapacityPeriod      `json:"capacity_period"`
	EfficiencyRate  float64                   `json:"efficiency_rate"`
	IsActive        bool                      `json:"is_active"`
	CreatedBy       uuid.UUID                 `json:"created_by"`
}

type UpdateMachineDTO struct {
	Code            int64                     `json:"code"`
	Name            string                    `json:"name"`
	MachineTypeCode int64                     `json:"machine_type_code"`
	CostCenterCode  *int64                    `json:"cost_center_code,omitempty"`
	Capacity        float64                   `json:"capacity"`
	CapacityUnit    types.MachineCapacityUnit `json:"capacity_per_unit"`
	CapacityPeriod  types.CapacityPeriod      `json:"capacity_period"`
	EfficiencyRate  float64                   `json:"efficiency_rate"`
	IsActive        bool                      `json:"is_active"`
	UpdatedBy       uuid.UUID                 `json:"updated_by"`
}

type CreateItemMachineTimeDTO struct {
	ItemCode           int64                `json:"item_code"`
	Mask               *string              `json:"mask,omitempty"`
	MachineCode        int64                `json:"machine_code"`
	ProductionTime     float64              `json:"production_time"`
	ProductionTimeUnit types.CapacityPeriod `json:"production_time_unit"`
	ProductionBaseQty  int                  `json:"production_base_qty"`
	SetupTime          float64              `json:"setup_time"`
	Priority           int                  `json:"priority"`
}

type CreateMachineScheduleDTO struct {
	MachineCode      int64   `json:"machine_code"`
	OrderCode        int64   `json:"order_code"`
	ScheduleDate     string  `json:"schedule_date"`
	StartTime        *string `json:"start_time,omitempty"`
	EndTime          *string `json:"end_time,omitempty"`
	PlannedQty       float64 `json:"planned_qty"`
	Sequence         int     `json:"sequence"`
	PriorityOverride *int    `json:"priority_override,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

type ReorderScheduleDTO struct {
	ScheduleCode     int64 `json:"schedule_id"`
	NewSequence      int   `json:"new_sequence"`
	PriorityOverride *int  `json:"priority_override,omitempty"`
}
