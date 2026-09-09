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
	ID              int64
	Code            int64
	Name            string
	MachineTypeCode int64
	CostCenterCode  *int64
	Capacity        float64
	CapacityUnit    types.MachineCapacityUnit
	CapacityPeriod  types.CapacityPeriod
	EfficiencyRate  float64
	IsActive        bool

	// Cadastro completo do recurso, no nível do que o mercado pede (FoccoERP
	// FENG0111): a que grupo e calendário a máquina pertence, onde fica, se é
	// gargalo, quando foi adquirida, quanto tempo leva para preparar, de quem
	// foi comprada e quem cuida da manutenção. As colunas já existiam na
	// tabela; faltava o caminho DTO → entidade → SQL.
	ResourceGroupID                  *int64
	CalendarID                       *int64
	Location                         *string
	IsCritical                       bool
	UsageDescription                 *string
	AcquiredOn                       *time.Time
	PreparationTime                  float64
	PreparationTimeUnit              string
	SupplierCode                     *int64
	Brand                            *string
	IsPreferred                      bool
	MaintenanceResponsibleEmployeeID *int64

	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy uuid.UUID
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
	OrderCode        *int64 // optional: a slot need not come from a planned order
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
