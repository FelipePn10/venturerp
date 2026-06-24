package response

import (
	"time"

	"github.com/google/uuid"
)

// MachineTypeResponse is the API representation of a machine type.
type MachineTypeResponse struct {
	ID               int64     `json:"id"`
	Code             int64     `json:"code"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	Type             string    `json:"type"`
	RequiresOperator bool      `json:"requires_operator"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedBy        uuid.UUID `json:"created_by"`
}

// MachineResponse is the API representation of a machine.
type MachineResponse struct {
	Code            int64     `json:"code"`
	Name            string    `json:"name"`
	MachineTypeCode int64     `json:"machine_type_code"`
	CostCenterCode  *int64    `json:"cost_center_code,omitempty"`
	Capacity        float64   `json:"capacity"`
	CapacityUnit    string    `json:"capacity_unit"`
	CapacityPeriod  string    `json:"capacity_period"`
	EfficiencyRate  float64   `json:"efficiency_rate"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       uuid.UUID `json:"created_by"`
}

// ItemMachineTimeResponse is the API representation of an item↔machine time config.
type ItemMachineTimeResponse struct {
	ItemCode           int64     `json:"item_code"`
	Mask               *string   `json:"mask,omitempty"`
	MachineCode        int64     `json:"machine_code"`
	ProductionTime     float64   `json:"production_time"`
	ProductionTimeUnit string    `json:"production_time_unit"`
	ProductionBaseQty  int       `json:"production_base_qty"`
	SetupTime          float64   `json:"setup_time"`
	Priority           int       `json:"priority"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// MachineScheduleResponse is the API representation of a machine schedule slot.
type MachineScheduleResponse struct {
	Code             int64      `json:"code"`
	MachineCode      int64      `json:"machine_code"`
	OrderCode        *int64     `json:"order_code,omitempty"`
	ScheduleDate     time.Time  `json:"schedule_date"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	PlannedQty       float64    `json:"planned_qty"`
	ProducedQty      float64    `json:"produced_qty"`
	Status           string     `json:"status"`
	Sequence         int        `json:"sequence"`
	PriorityOverride *int       `json:"priority_override,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
