package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type CreateMachineTypeDTO struct {
	Code             int64                 `json:"code"`
	Name             string                `json:"name"`
	Description      *string               `json:"description,omitempty"`
	Type             types.MachineTypeEnum `json:"type"`
	RequiresOperator bool                  `json:"requires_operator"`
	// CreatedBy vem do JWT; nunca do corpo da requisição.
	CreatedBy uuid.UUID `json:"-"`
	// IsActive é ponteiro para distinguir "não informado" de "false". Sem isso o
	// centro de trabalho nascia inativo e sumia de todas as consultas — e a
	// criação de máquina o recusava com um erro que não explicava o motivo.
	IsActive *bool `json:"is_active,omitempty"`
}

// AtivoOuPadrao devolve o valor informado ou `true`, que é o esperado ao criar.
func (d CreateMachineTypeDTO) AtivoOuPadrao() bool {
	if d.IsActive == nil {
		return true
	}
	return *d.IsActive
}

type UpdateMachineTypeDTO struct {
	Code             int64                 `json:"code"`
	Name             string                `json:"name"`
	Description      *string               `json:"description,omitempty"`
	Type             types.MachineTypeEnum `json:"type"`
	RequiresOperator bool                  `json:"requires_operator"`
	IsActive         bool                  `json:"is_active"`
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
	// Mesmo motivo do tipo de máquina: omitir passa a significar "ativa".
	IsActive *bool `json:"is_active,omitempty"`
	// CreatedBy vem do JWT; nunca do corpo da requisição.
	CreatedBy uuid.UUID `json:"-"`
}

func (d CreateMachineDTO) AtivoOuPadrao() bool {
	if d.IsActive == nil {
		return true
	}
	return *d.IsActive
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
	// UpdatedBy vem do JWT; nunca do corpo da requisição.
	UpdatedBy uuid.UUID `json:"-"`
}

type CreateItemMachineTimeDTO struct {
	ItemCode           TextCode             `json:"item_code"`
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
