package response

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

// ProductionOrderResponse is the API representation of a production order (OF).
type ProductionOrderResponse struct {
	// CostPending: OF encerrada sem custo real apurado (motivo).
	CostPending *string `json:"cost_pending,omitempty"`
	// ItemRevision: a revisão do item quando a OF nasceu.
	ItemRevision   *string    `json:"item_revision,omitempty"`
	ID             int64      `json:"id"`
	OrderNumber    int64      `json:"order_number"`
	PlannedOrderID *int64     `json:"planned_order_id,omitempty"`
	ItemCode       int64      `json:"item_code"`
	Mask           string     `json:"mask"`
	PlannedQty     float64    `json:"planned_qty"`
	ProducedQty    float64    `json:"produced_qty"`
	ScrappedQty    float64    `json:"scrapped_qty"`
	Status         string     `json:"status"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	MachineID      *int64     `json:"machine_id,omitempty"`
	CostCenterID   *int64     `json:"cost_center_id,omitempty"`
	EmployeeID     *int64     `json:"employee_id,omitempty"`
	Priority       *string    `json:"priority,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	// Origin: cliente, projeto, pedido/demanda e lugar na cadeia (OF pai e
	// produto final) — para reconhecer a OF sem decorar o número.
	Origin *ProductionOrderOrigin `json:"origin,omitempty"`
	// Warnings: avisos do início da OF (componente sem saldo, OF filha aberta)
	// quando o parâmetro de planejamento 52 está em AVISAR.
	Warnings []string `json:"warnings,omitempty"`
}

// ProductionAppointmentResponse is the API representation of a production appointment.
type ProductionAppointmentResponse struct {
	ID                int64     `json:"id"`
	ProductionOrderID int64     `json:"production_order_id"`
	MachineID         *int64    `json:"machine_id,omitempty"`
	EmployeeID        *int64    `json:"employee_id,omitempty"`
	AppointmentDate   time.Time `json:"appointment_date"`
	StartTime         *string   `json:"start_time,omitempty"`
	EndTime           *string   `json:"end_time,omitempty"`
	ProducedQty       float64   `json:"produced_qty"`
	ScrappedQty       float64   `json:"scrapped_qty"`
	ScrapReason       *string   `json:"scrap_reason,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

// ProductionConsumptionResponse is the API representation of a production consumption.
type ProductionConsumptionResponse struct {
	ID                int64     `json:"id"`
	ProductionOrderID int64     `json:"production_order_id"`
	AppointmentID     *int64    `json:"appointment_id,omitempty"`
	ItemCode          int64     `json:"item_code"`
	ConsumedQty       float64   `json:"consumed_qty"`
	WarehouseID       *int64    `json:"warehouse_id,omitempty"`
	Lot               *string   `json:"lot,omitempty"`
	ConsumptionDate   time.Time `json:"consumption_date"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

type ProductionDeliveryResponse struct {
	ID            int64           `json:"id"`
	Quantity      decimal.Decimal `json:"quantity"`
	MovementClass string          `json:"movement_class"`
	WarehouseID   int64           `json:"warehouse_id"`
	Lot           *string         `json:"lot,omitempty"`
	Final         bool            `json:"final"`
	DeliveredAt   time.Time       `json:"delivered_at"`
}

type ProductionMovementResponse struct {
	ID           int64   `json:"id"`
	ItemCode     int64   `json:"item_code"`
	WarehouseID  int64   `json:"warehouse_id"`
	MovementType string  `json:"movement_type"`
	Quantity     float64 `json:"quantity"`
	Lot          *string `json:"lot,omitempty"`
}

type ProductionOrderOperationalResponse struct {
	Order        *ProductionOrderResponse         `json:"order"`
	Deliveries   []*ProductionDeliveryResponse    `json:"deliveries"`
	Appointments []*ProductionAppointmentResponse `json:"appointments"`
	Consumptions []*ProductionConsumptionResponse `json:"consumptions"`
	Movements    []*ProductionMovementResponse    `json:"movements"`
	Totals       map[string]decimal.Decimal       `json:"totals"`
}

// ProductionOrderCostResponse é o fechamento de custo da ordem: o que a ordem
// realmente custou, o que deveria ter custado pelo padrão, e a diferença — em
// material, mão de obra e indiretos.
//
// Antes a entidade de domínio era serializada direto, sem tags: os campos
// chegavam ao cliente em PascalCase (`MaterialCostReal`) e qualquer renomeação
// interna quebrava o contrato sem aviso. É a valorização de ordem que o
// FoccoERP tem no FCST0206 e o SAP no encerramento da ordem.
type ProductionOrderCostResponse struct {
	ProductionOrderID int64   `json:"production_order_id"`
	ProducedQty       float64 `json:"produced_qty"`

	MaterialCostReal float64 `json:"material_cost_real"`
	// Máquina = preparação + máquina; terceiros = serviço externo recebido.
	MachineCostReal     float64 `json:"machine_cost_real"`
	SubcontractCostReal float64 `json:"subcontract_cost_real"`
	LaborCostReal       float64 `json:"labor_cost_real"`
	OverheadCostReal    float64 `json:"overhead_cost_real"`
	TotalCostReal       float64 `json:"total_cost_real"`
	UnitCostReal        float64 `json:"unit_cost_real"`

	MaterialCostStd    float64 `json:"material_cost_std"`
	MachineCostStd     float64 `json:"machine_cost_std"`
	SubcontractCostStd float64 `json:"subcontract_cost_std"`
	LaborCostStd       float64 `json:"labor_cost_std"`
	OverheadCostStd    float64 `json:"overhead_cost_std"`
	TotalCostStd       float64 `json:"total_cost_std"`

	MaterialVariance    float64 `json:"material_variance"`
	MachineVariance     float64 `json:"machine_variance"`
	SubcontractVariance float64 `json:"subcontract_variance"`
	LaborVariance       float64 `json:"labor_variance"`
	OverheadVariance    float64 `json:"overhead_variance"`
	TotalVariance       float64 `json:"total_variance"`
	// VariancePct é a diferença em porcentagem do padrão. É por ela que o
	// custo olha primeiro: R$ 500 de desvio pesam diferente numa ordem de
	// R$ 2.000 e numa de R$ 200.000.
	VariancePct float64 `json:"variance_pct"`

	Currency  string `json:"currency"`
	SettledAt string `json:"settled_at,omitempty"`
}

// ProductionChainResponse é a cadeia de fabricação da OF: pedidos de venda,
// ordens pai que ela abastece e ordens filhas dos componentes fabricados.
type ProductionChainResponse struct {
	ProductionOrderID int64                       `json:"production_order_id"`
	SalesOrders       []ProductionChainSalesOrder `json:"sales_orders"`
	// IndependentDemands: demandas independentes (sem pedido de venda) que a
	// OF atende, direto ou pela OF pai.
	IndependentDemands []ProductionChainDemand `json:"independent_demands"`
	Parents            []ProductionChainOrder  `json:"parents"`
	Children           []ProductionChainOrder  `json:"children"`
}

type ProductionChainSalesOrder struct {
	SalesOrderCode     int64  `json:"sales_order_code"`
	SalesOrderNumber   int64  `json:"sales_order_number"`
	SalesOrderItemCode *int64 `json:"sales_order_item_code,omitempty"`
	Sequence           *int32 `json:"sequence,omitempty"`
	Customer           string `json:"customer"`
}

type ProductionChainDemand struct {
	DemandCode   int64           `json:"demand_code"`
	Reason       string          `json:"reason"`
	ReasonLabel  string          `json:"reason_label"`
	ItemCode     int64           `json:"item_code"`
	BusinessCode string          `json:"business_code,omitempty"`
	Description  string          `json:"description"`
	Quantity     decimal.Decimal `json:"quantity"`
	DemandDate   string          `json:"demand_date"`
	Status       string          `json:"status"`
}

type ProductionChainOrder struct {
	PlannedOrderCode  *int64          `json:"planned_order_code,omitempty"`
	ProductionOrderID *int64          `json:"production_order_id,omitempty"`
	OrderNumber       *int64          `json:"order_number,omitempty"`
	ItemCode          int64           `json:"item_code"`
	BusinessCode      string          `json:"business_code,omitempty"`
	Description       string          `json:"description"`
	Status            string          `json:"status"`
	Quantity          decimal.Decimal `json:"quantity"`
	Produced          decimal.Decimal `json:"produced"`
	Reserved          decimal.Decimal `json:"reserved"`
}

// ProductionOrderOrigin identifica a OF: de quem, para quê e para qual OF.
type ProductionOrderOrigin struct {
	ItemBusinessCode string  `json:"item_business_code"`
	ItemName         string  `json:"item_name"`
	CustomerCode     *int64  `json:"customer_code,omitempty"`
	CustomerName     string  `json:"customer_name,omitempty"`
	ProjectID        *int64  `json:"project_id,omitempty"`
	ProjectCode      string  `json:"project_code,omitempty"`
	ProjectName      string  `json:"project_name,omitempty"`
	SalesOrders      []int64 `json:"sales_orders"`
	Demands          []int64 `json:"demands"`
	// ParentOrderID: OF pai (id); ParentPlannedOrder: a ordem planejada pai
	// quando ela ainda não virou OF.
	ParentOrderID      *int64 `json:"parent_order_id,omitempty"`
	ParentOrderNumber  *int64 `json:"parent_order_number,omitempty"`
	ParentPlannedOrder *int64 `json:"parent_planned_order,omitempty"`
	ParentItem         string `json:"parent_item,omitempty"`
	RootOrderID        *int64 `json:"root_order_id,omitempty"`
	RootOrderNumber    *int64 `json:"root_order_number,omitempty"`
	RootItem           string `json:"root_item,omitempty"`
	Level              int    `json:"level"`
	Children           int    `json:"children"`
}
