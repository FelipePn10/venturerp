package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProductionOrderStatus string

const (
	StatusOpen       ProductionOrderStatus = "OPEN"
	StatusInProgress ProductionOrderStatus = "IN_PROGRESS"
	StatusCompleted  ProductionOrderStatus = "COMPLETED"
	StatusClosed     ProductionOrderStatus = "CLOSED"
	StatusCancelled  ProductionOrderStatus = "CANCELLED"
)

type ProductionOrder struct {
	ID             int64
	OrderNumber    int64
	PlannedOrderID *int64
	ItemCode       int64
	Mask           string
	PlannedQty     float64
	ProducedQty    float64
	ScrappedQty    float64
	Status         ProductionOrderStatus
	StartDate      *time.Time
	EndDate        *time.Time
	MachineID      *int64
	CostCenterID   *int64
	EmployeeID     *int64
	WarehouseID    *int64
	Priority       *string
	Notes          *string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
}

type ProductionAppointment struct {
	ID                int64
	ProductionOrderID int64
	MachineID         *int64
	EmployeeID        *int64
	AppointmentDate   time.Time
	StartTime         *string
	EndTime           *string
	ProducedQty       float64
	ScrappedQty       float64
	ScrapReason       *string
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         uuid.UUID
}

type ProductionConsumption struct {
	ID                int64
	ProductionOrderID int64
	AppointmentID     *int64
	ItemCode          int64
	ConsumedQty       float64
	WarehouseID       *int64
	Lot               *string
	ConsumptionDate   time.Time
	Notes             *string
	CreatedAt         time.Time
	CreatedBy         uuid.UUID
}

type ProductionDelivery struct {
	ID                int64
	ProductionOrderID int64
	Quantity          decimal.Decimal
	IdempotencyKey    string
	MovementClass     string
	WarehouseID       int64
	Lot               *string
	IsFinal           bool
	DeliveredAt       time.Time
	CreatedBy         uuid.UUID
	Lines             []ProductionDeliveryLine
}

type ProductionDeliveryLine struct {
	MovementClass string
	Quantity      decimal.Decimal
}
