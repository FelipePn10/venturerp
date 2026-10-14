package entity

import (
	"time"

	"github.com/google/uuid"
)

type MRPItemProfile struct {
	ItemCode        int64
	PlanCode        int64
	CalculationDate time.Time
	Demand          float64
	OrdersPlanned   float64
	OrdersFirm      float64
	StockProjected  float64
	LLC             int
	NeedDate        time.Time
	CreatedAt       time.Time
}

type MRPProfileDetail struct {
	PlanCode       int64
	ItemCode       int64
	NeedDate       time.Time
	DetailType     string
	SourceCode     *int64
	ParentItemCode *int64
	Quantity       float64
}

type MRPCalculationLog struct {
	Code        int64
	PlanCode    int64
	StartedAt   time.Time
	FinishedAt  *time.Time
	Status      string
	Errors      map[string]interface{}
	TotalItems  int
	TotalOrders int
	CreatedAt   time.Time
}

type StockSnapshot struct {
	ItemCode      int64
	WarehouseCode int64
	Quantity      float64
	ReservedQty   float64
	SafetyStock   float64
	SnapshotDate  time.Time
	CreatedAt     time.Time
}

type SalesOrderDemand struct {
	Code           int64
	SalesOrderCode int64
	ItemCode       int64
	Mask           *string
	Quantity       float64
	DeliveredQty   float64
	DeliveryDate   time.Time
	DivisionCode   *int64
	Status         string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ConfiguredItemRule struct {
	Code      int64
	ItemCode  int64
	TableType string
	FieldName string
	RuleType  string
	RuleValue string
	Sequence  int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy uuid.UUID
}

// MRPInput holds all inputs for MRP calculation
type MRPInput struct {
	PlanCode             int64
	ItemCode             int64
	Mask                 string
	Quantity             float64
	NeedDate             time.Time
	SalesOrderCode       *int64
	ParentItemCode       *int64
	LLC                  int
	DemandType           string
	SourceCode           *int64
	WarehouseCode        *int64
	TechnicalAssistance  bool
	InterFactory         bool
	SourceEnterpriseCode *int64
	AutoRelease          bool
	// PegKind/PegCode identificam a origem da necessidade para o pegging
	// quando DemandType/SourceCode não bastam: SUGESTAO (sugestão do pai que
	// gerou esta necessidade), MPS, ESTOQUE_SEGURANCA, SIMULADA.
	PegKind string
	PegCode *int64
}

// MRPOutput holds the result of MRP calculation for one item
type MRPOutput struct {
	ItemCode       int64
	LLC            int
	Demand         float64
	StockProjected float64
	NetRequirement float64
	PlannedOrders  []*PlannedOrderSuggestion
}

// PlannedOrderSuggestion é devolvida direto pelo handler de sugestões, então as
// tags são o contrato da API. Sem elas os campos saíam em PascalCase e o
// middleware de código de item não reconhecia `ItemCode` — a lista do MRP
// mostrava a chave interna (10) no lugar do código que o usuário conhece
// (MP-CH-3MM), e o mesmo valor ia para a ordem firmada.
type PlannedOrderSuggestion struct {
	Code                 int64      `json:"code"`
	OrderNumber          *int64     `json:"order_number"`
	WarehouseCode        *int64     `json:"warehouse_code"`
	InterFactory         bool       `json:"inter_factory"`
	SourceEnterpriseCode *int64     `json:"source_enterprise_code"`
	AutoRelease          bool       `json:"auto_release"`
	PlanCode             int64      `json:"plan_code"`
	ItemCode             int64      `json:"item_code"`
	Mask                 string     `json:"mask"`
	Quantity             float64    `json:"quantity"`
	NeedDate             time.Time  `json:"need_date"`
	StartDate            *time.Time `json:"start_date"`
	OrderType            string     `json:"order_type"`
	DemandType           string     `json:"demand_type"`
	ParentItemCode       *int64     `json:"parent_item_code"`
	LLC                  int        `json:"llc"`
	MachineID            *int64     `json:"machine_id"`
	MachineCode          *int64     `json:"machine_code,omitempty"`
	EstimatedEndAt       *time.Time `json:"estimated_end_at,omitempty"`
	RequestedStartDate   *time.Time `json:"-"`
	CapacityLate         bool       `json:"capacity_late"`
	ProductionTime       *float64   `json:"production_time"`
	Priority             *string    `json:"priority"`
	Notes                *string    `json:"notes"`
	RouteOperationID     *int64     `json:"route_operation_id"`
	OperationID          *int64     `json:"operation_id"`
	SupplierCode         *int64     `json:"supplier_code"`
	ServiceItemCode      *int64     `json:"service_item_code"`
	RemittanceType       *string    `json:"remittance_type"`
	// PlannedOrderCode é a ordem planejada que a sugestão gerou ao ser firmada.
	// Firmar de novo devolve a mesma ordem; sem este campo a tela não sabia o
	// que já tinha sido firmado e contava tudo como pendente.
	PlannedOrderCode *int64 `json:"planned_order_code,omitempty"`
}

// ExceptionMessageType classifies an MRP exception so planners can act
// on it without reading the description text.
type ExceptionMessageType string

const (
	// ExceptionRescheduleIn — an existing firm order arrives after the demand
	// need date. The planner must expedite or reschedule it earlier.
	ExceptionRescheduleIn ExceptionMessageType = "RESCHEDULE_IN"

	// ExceptionRescheduleOut — an existing firm order arrives well before the
	// demand need date (>30 days). Delaying it would free up cash and storage.
	ExceptionRescheduleOut ExceptionMessageType = "RESCHEDULE_OUT"

	// ExceptionCancel — an existing firm order has no covering demand in this
	// MRP run. Cancelling it avoids unnecessary production or purchase.
	ExceptionCancel ExceptionMessageType = "CANCEL"

	// ExceptionExpedite — demand is urgent and an existing order must be
	// accelerated beyond its normal lead time.
	ExceptionExpedite ExceptionMessageType = "EXPEDITE"

	// ExceptionExcess — total firm supply significantly exceeds the net
	// requirement. Projected excess stock after all demands are satisfied.
	ExceptionExcess ExceptionMessageType = "EXCESS_PROJECTED"
)

// MRPExceptionMessage is generated by the MRP engine when it detects that an
// existing firm order is misaligned with current demand. Each message is
// actionable — the planner knows exactly what to do without needing to run
// a manual analysis.
type MRPExceptionMessage struct {
	Code        int64
	PlanCode    int64
	ItemCode    int64
	MessageType ExceptionMessageType
	SourceCode  *int64  // PK of the planned_order that triggered this (nil = demand-driven)
	SourceType  *string // e.g. "PLANNED_ORDER", "PURCHASE_ORDER"
	Description string  // human-readable explanation in Brazilian Portuguese
	CreatedAt   time.Time
	// SuggestedDate é a data em que o cálculo precisa do suprimento: reprogramar
	// a ordem a partir da exceção leva a ela.
	SuggestedDate *time.Time
	// Tratamento registrado pelo planejador (nil = pendente).
	HandledAt      *time.Time
	HandledBy      *string
	HandledByName  string
	HandlingAction string
	HandlingNote   string
	// Ordem planejada de origem, quando a exceção vem de uma: número e
	// necessidade atual.
	SourceOrderNumber *int64
	SourceNeedDate    *time.Time
}

// Como o planejador tratou uma exceção do MRP.
const (
	ExceptionHandledRescheduled  = "REPROGRAMADA"
	ExceptionHandledCancelled    = "CANCELADA"
	ExceptionHandledAcknowledged = "TRATADA"
)

// ---------- New entity types for enhanced MRP features ----------

// MachineTimeInfo holds the relationship between an item, a machine, and its
// unit production time. Used for machine scheduling during MRP.
type MachineTimeInfo struct {
	WorkCenterID          int64
	Mask                  string
	MachineCode           int64
	ProductionTimeUnit    string
	ProductionBaseQty     int
	SetupTime             float64
	EfficiencyRate        *float64
	MachineEfficiencyRate float64
	WorkingHoursPerDay    float64
	TimeBasis             string
	// Consumível: taxa por hora de usinagem deste item, autonomia de uma carga e
	// tempo de troca. Nulos quando o item não declara consumo — o caminho sem
	// consumível continua idêntico.
	ConsumptionPerHour    *float64
	ConsumableCapacity    *float64
	ConsumableSwapMinutes *float64
	ConsumableUnit        *string
	ConsumableName        *string
	ItemCode              int64
	MachineID             int64
	Priority              int // lower = higher importance
	ProductionTime        float64
}

// KanbanCardInfo represents an active kanban card tied to an item.
type KanbanCardInfo struct {
	CardCode        int64
	ItemCode        int64
	ReorderPoint    float64
	QuantityPerCard float64
	CardCount       int
	IsActive        bool
}

// MPSItemInfo is a single entry from the Master Production Schedule.
type MPSItemInfo struct {
	ItemCode    int64
	Mask        string
	PeriodType  string
	PeriodValue int
	Year        int
	Quantity    float64
	IsFirm      bool
}

// ItemPlanningExtra holds extra planning parameters per item that are not
// part of the main item record (e.g. maximum stock level for MIN_MAX).
type ItemPlanningExtra struct {
	ItemCode     int64
	MaximumStock float64
	SafetyTime   int
	Coverage     int
	GroupingKey  *string
	IsCritical   bool
	UseTankDate  bool
}

// MachineScheduleInfo is a machine allocation record generated by MRP
// for FABRICACAO-type planned orders.
type MachineScheduleInfo struct {
	PlanCode         int64
	PlannedOrderCode int64
	MachineID        int64
	ScheduleDate     time.Time
	ProductionTime   float64
}

// ── Pegging ─────────────────────────────────────────────────────────────────

// Origem de uma necessidade no pegging.
const (
	PegDemandSalesOrder  = "PEDIDO_VENDA"
	PegDemandIndependent = "DEMANDA_INDEPENDENTE"
	PegDemandForecast    = "PREVISAO"
	PegDemandMPS         = "MPS"
	PegDemandSafetyStock = "ESTOQUE_SEGURANCA"
	PegDemandSuggestion  = "SUGESTAO"
	PegDemandSimulated   = "SIMULADA"
	PegDemandOther       = "OUTRA"
)

// Suprimento que cobriu a necessidade.
const (
	PegSupplyStock      = "ESTOQUE"
	PegSupplyFirmOrder  = "ORDEM_FIRME"
	PegSupplySuggestion = "SUGESTAO"
	PegSupplyShortage   = "FALTA"
)

// PeggingRecord diz que parte de uma necessidade foi coberta por qual
// suprimento no cálculo do plano.
type PeggingRecord struct {
	ItemCode   int64
	DemandKind string
	DemandCode *int64
	NeedDate   time.Time
	SupplyKind string
	SupplyCode *int64
	SupplyDate *time.Time
	Quantity   float64
}

// ── Cenários de simulação ───────────────────────────────────────────────────

// Referência de um ajuste de demanda do cenário.
const (
	ScenarioDemandNew         = "NOVA"
	ScenarioDemandIndependent = "DEMANDA_INDEPENDENTE"
	ScenarioDemandSalesOrder  = "PEDIDO_VENDA"
)

// ScenarioDemandChange altera, retira (quantidade 0) ou cria uma demanda no
// cálculo do cenário.
type ScenarioDemandChange struct {
	ID       int64
	RefKind  string
	RefCode  *int64
	ItemCode int64
	Mask     string
	Quantity *float64
	NeedDate *time.Time
}

// ScenarioAdjustments são os ajustes que o motor do MRP aplica em memória ao
// calcular um plano de cenário. A capacidade de máquina não está aqui: ela
// entra nas consultas de janela e de capacidade, que leem o cenário do plano.
type ScenarioAdjustments struct {
	Demands []ScenarioDemandChange
	// StockDelta é a entrada (+) ou baixa (−) simulada por item.
	StockDelta map[int64]float64
	// LeadTimeDays substitui o prazo do item.
	LeadTimeDays map[int64]int
}

// PeggingRefs são os códigos cujo rótulo a consulta do pegging precisa.
type PeggingRefs struct {
	Suggestions     []int64
	FirmOrders      []int64
	SalesOrderItems []int64
	Items           []int64
}

type PeggingOrderLabel struct {
	OrderNumber *int64
	OrderType   string
	Quantity    float64
	ItemCode    int64
	StartDate   *time.Time
	NeedDate    *time.Time
	// EstimatedEnd é o fim programado na máquina (sugestão de fabricação com
	// capacidade finita); nil quando não passou pela programação.
	EstimatedEnd *time.Time
	// EstimatedEndDay é o dia desse fim no fuso do banco (o da fábrica) —
	// para exibir sem que a conversão de horário troque o dia.
	EstimatedEndDay *time.Time
}

type PeggingSalesLabel struct {
	OrderNumber *int64
	Customer    string
}

type PeggingItemLabel struct {
	BusinessCode string
	Description  string
	UOM          string
}

type PeggingLabels struct {
	Suggestions     map[int64]PeggingOrderLabel
	FirmOrders      map[int64]PeggingOrderLabel
	SalesOrderItems map[int64]PeggingSalesLabel
	Items           map[int64]PeggingItemLabel
}

// SalesOrderPlan é um plano cujo último cálculo considerou itens do pedido de
// venda — o ponto de partida do "atendimento" do pedido.
type SalesOrderPlan struct {
	PlanCode     int64      `json:"plan_code"`
	PlanName     string     `json:"plan_name"`
	CalculatedAt *time.Time `json:"calculated_at,omitempty"`
	ItemCodes    []int64    `json:"-"`
}
