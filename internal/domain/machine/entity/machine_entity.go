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
	InheritWorkCenterHours bool
	AvailableHoursPerDay   *float64
	ID                     int64
	Code                   int64
	Name                   string
	MachineTypeCode        int64
	CostCenterCode         *int64
	Capacity               float64
	CapacityUnit           types.MachineCapacityUnit
	CapacityPeriod         types.CapacityPeriod
	EfficiencyRate         float64
	IsActive               bool

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

// MachineConsumable é o que a máquina gasta enquanto produz — gás de corte,
// eletrodo, arame, óleo. Guarda a AUTONOMIA (quanto rende uma carga) e o tempo
// de troca; a taxa de consumo não mora aqui porque depende do que está sendo
// feito, e por isso fica em ItemMachineTime.
type MachineConsumable struct {
	ID                 int64
	MachineCode        int64
	Code               string
	Description        string
	Unit               string
	CapacityPerRefill  float64
	ReplacementMinutes float64
	IsActive           bool
}

// ConsumableUsage é a autonomia do consumível que este item gasta na máquina.
// O consumo depende do que está sendo feito — uma chapa de 3 mm e uma de 12 mm
// gastam vazões diferentes de gás na mesma máquina —, por isso a TAXA vive aqui,
// no par item × máscara × máquina, e não no cadastro da máquina.
type ConsumableUsage struct {
	// PerHour é o consumo por hora de USINAGEM, na unidade do consumível.
	// Preparação não corta, então não consome.
	PerHour float64
	// CapacityPerRefill é quanto rende uma carga completa (ex.: 200 m³).
	CapacityPerRefill float64
	// ReplacementMinutes é quanto a máquina fica parada para trocar a carga.
	ReplacementMinutes float64
	Unit               string
	Description        string
}

type ItemMachineTime struct {
	EfficiencyRate *float64
	// Consumable é a autonomia resolvida (usada no cálculo). ConsumableID e
	// ConsumptionPerHour são o que se grava: o vínculo e a taxa.
	Consumable         *ConsumableUsage
	ConsumableID       *int64
	ConsumptionPerHour *float64
	TimeBasis          string
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
