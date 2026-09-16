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
	Code        int64                 `json:"code"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	Type        types.MachineTypeEnum `json:"type"`
	// Ponteiro pelo mesmo motivo de IsActive: com `bool` puro, omitir o campo é
	// indistinguível de mandar false, então quem gravasse um corpo parcial
	// desligava a flag em silêncio. Nulo = mantém o valor atual; false explícito
	// continua desligando.
	RequiresOperator *bool `json:"requires_operator,omitempty"`
	IsActive         *bool `json:"is_active,omitempty"`
}

type CreateMachineDTO struct {
	AvailableHoursPerDay *float64                  `json:"available_hours_per_day,omitempty"`
	Code                 int64                     `json:"code"`
	Name                 string                    `json:"name"`
	MachineTypeCode      int64                     `json:"machine_type_code"`
	CostCenterCode       *int64                    `json:"cost_center_code,omitempty"`
	Capacity             float64                   `json:"capacity"`
	CapacityUnit         types.MachineCapacityUnit `json:"capacity_per_unit"`
	CapacityPeriod       types.CapacityPeriod      `json:"capacity_period"`
	EfficiencyRate       float64                   `json:"efficiency_rate"`
	// Mesmo motivo do tipo de máquina: omitir passa a significar "ativa".
	IsActive *bool `json:"is_active,omitempty"`

	// ─── Cadastro completo do recurso (FoccoERP FENG0111) ──────────────────
	// Grupo de recursos e calendário definem como o sequenciamento enxerga a
	// máquina; `is_critical` marca o gargalo; `is_preferred` decide qual
	// recurso é alocado primeiro quando a operação aceita mais de um.
	ResourceGroupID                  *int64   `json:"resource_group_id,omitempty"`
	CalendarID                       *int64   `json:"calendar_id,omitempty"`
	Location                         *string  `json:"location,omitempty"`
	IsCritical                       *bool    `json:"is_critical,omitempty"`
	UsageDescription                 *string  `json:"usage_description,omitempty"`
	AcquiredOn                       *string  `json:"acquired_on,omitempty"` // AAAA-MM-DD
	PreparationTime                  *float64 `json:"preparation_time,omitempty"`
	PreparationTimeUnit              *string  `json:"preparation_time_unit,omitempty"`
	SupplierCode                     *int64   `json:"supplier_code,omitempty"`
	Brand                            *string  `json:"brand,omitempty"`
	IsPreferred                      *bool    `json:"is_preferred,omitempty"`
	MaintenanceResponsibleEmployeeID *int64   `json:"maintenance_responsible_employee_id,omitempty"`
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
	InheritWorkCenterHours bool                      `json:"inherit_work_center_hours,omitempty"`
	AvailableHoursPerDay   *float64                  `json:"available_hours_per_day,omitempty"`
	Code                   int64                     `json:"code"`
	Name                   string                    `json:"name"`
	MachineTypeCode        int64                     `json:"machine_type_code"`
	CostCenterCode         *int64                    `json:"cost_center_code,omitempty"`
	Capacity               float64                   `json:"capacity"`
	CapacityUnit           types.MachineCapacityUnit `json:"capacity_per_unit"`
	CapacityPeriod         types.CapacityPeriod      `json:"capacity_period"`
	EfficiencyRate         float64                   `json:"efficiency_rate"`
	IsActive               *bool                     `json:"is_active,omitempty"`

	// ─── Cadastro completo do recurso (FoccoERP FENG0111) ──────────────────
	// Grupo de recursos e calendário definem como o sequenciamento enxerga a
	// máquina; `is_critical` marca o gargalo; `is_preferred` decide qual
	// recurso é alocado primeiro quando a operação aceita mais de um.
	ResourceGroupID                  *int64   `json:"resource_group_id,omitempty"`
	CalendarID                       *int64   `json:"calendar_id,omitempty"`
	Location                         *string  `json:"location,omitempty"`
	IsCritical                       *bool    `json:"is_critical,omitempty"`
	UsageDescription                 *string  `json:"usage_description,omitempty"`
	AcquiredOn                       *string  `json:"acquired_on,omitempty"` // AAAA-MM-DD
	PreparationTime                  *float64 `json:"preparation_time,omitempty"`
	PreparationTimeUnit              *string  `json:"preparation_time_unit,omitempty"`
	SupplierCode                     *int64   `json:"supplier_code,omitempty"`
	Brand                            *string  `json:"brand,omitempty"`
	IsPreferred                      *bool    `json:"is_preferred,omitempty"`
	MaintenanceResponsibleEmployeeID *int64   `json:"maintenance_responsible_employee_id,omitempty"`
	// UpdatedBy vem do JWT; nunca do corpo da requisição.
	UpdatedBy uuid.UUID `json:"-"`
}

type CreateItemMachineTimeDTO struct {
	EfficiencyRate     *float64             `json:"efficiency_rate,omitempty"`
	TimeBasis          string               `json:"time_basis,omitempty"`
	ItemCode           TextCode             `json:"item_code"`
	Mask               *string              `json:"mask,omitempty"`
	MachineCode        int64                `json:"machine_code"`
	ProductionTime     float64              `json:"production_time"`
	ProductionTimeUnit types.CapacityPeriod `json:"production_time_unit"`
	ProductionBaseQty  int                  `json:"production_base_qty"`
	SetupTime          float64              `json:"setup_time"`
	Priority           int                  `json:"priority"`
	// Consumível gasto por este item nesta máquina e a taxa por hora de usinagem.
	// Os dois andam juntos: taxa sem consumível não diz o que se gasta.
	ConsumableID       *int64   `json:"consumable_id,omitempty"`
	ConsumptionPerHour *float64 `json:"consumption_per_hour,omitempty"`
}

// MachineConsumableDTO é o cadastro do consumível na máquina: quanto rende uma
// carga e quanto a máquina fica parada para trocá-la.
type MachineConsumableDTO struct {
	MachineCode        int64   `json:"machine_code"`
	Code               string  `json:"code"`
	Description        string  `json:"description"`
	Unit               string  `json:"unit"`
	CapacityPerRefill  float64 `json:"capacity_per_refill"`
	ReplacementMinutes float64 `json:"replacement_minutes"`
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
