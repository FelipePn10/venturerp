package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type MachineType struct {
	ID               int64
	Code             int64
	Name             string
	Description      *string
	Type             types.MachineTypeEnum
	RequiresOperator bool // true = operador humano controla a máquina; overlap ignorado no CPM
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CreatedBy        uuid.UUID
}

type Machine struct {
	Code            int64
	Name            string
	MachineTypeCode int64
	CostCenterCode  *int64
	Capacity        float64
	CapacityUnit    types.MachineCapacityUnit
	CapacityPeriod  types.CapacityPeriod
	EfficiencyRate  float64
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       uuid.UUID
}

type ItemMachineTime struct {
	ItemCode           int64
	Mask               *string
	MachineCode        int64
	ProductionTime     float64              // 5
	ProductionTimeUnit types.CapacityPeriod // minutos
	ProductionBaseQty  int                  // para fazer 1 item
	SetupTime          float64
	Priority           int
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type MachineSchedule struct {
	Code             int64
	MachineCode      int64
	OrderCode        int64
	ScheduleDate     time.Time
	StartTime        *time.Time
	EndTime          *time.Time
	PlannedQty       float64
	ProducedQty      float64
	Status           string
	Sequence         int
	PriorityOverride *int
	Notes            *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
