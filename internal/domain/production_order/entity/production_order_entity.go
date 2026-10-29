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
	PlannedQty     decimal.Decimal
	ProducedQty    decimal.Decimal
	ScrappedQty    decimal.Decimal
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
	// CostPending: a OF foi encerrada, mas a apuração do custo real falhou
	// (motivo). Vazio quando o custo foi apurado.
	CostPending *string
	// VinculoDemanda liga a OF nova à demanda independente que ela cobre,
	// na mesma transação da criação ("Gerar OF agora" da VPLA0102). Só na
	// criação; a leitura não o preenche.
	VinculoDemanda *VinculoDemanda
}

// VinculoDemanda é a demanda que a OF cobre e quanto dela. OrdensVistas é
// quantas ordens a demanda tinha quando a quantidade foi decidida: se outra
// ordem foi ligada nesse meio tempo, a criação é recusada (sem OF em dobro).
type VinculoDemanda struct {
	DemandID     int64
	Quantidade   decimal.Decimal
	OrdensVistas int
}

type ProductionAppointment struct {
	ID                int64
	ProductionOrderID int64
	MachineID         *int64
	EmployeeID        *int64
	AppointmentDate   time.Time
	StartTime         *string
	EndTime           *string
	ProducedQty       decimal.Decimal
	ScrappedQty       decimal.Decimal
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
	ConsumedQty       decimal.Decimal
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
	// ItemCode/Mask: o item entregue, para criar as séries na mesma transação.
	ItemCode int64
	Mask     *string
	// NumerosSerie informadas na entrega (item que controla série).
	NumerosSerie []string
	// SeriesGravadas: as séries criadas (informadas ou geradas pela máscara).
	SeriesGravadas []string
}

type ProductionDeliveryLine struct {
	MovementClass string
	Quantity      decimal.Decimal
}
