package response

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

// ProductionOrderResponse is the API representation of a production order (OF).
type ProductionOrderResponse struct {
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
