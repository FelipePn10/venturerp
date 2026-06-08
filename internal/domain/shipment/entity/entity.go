package entity

import (
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	ShipmentStatusOpen      ShipmentStatus = "OPEN"
	ShipmentStatusSeparated ShipmentStatus = "SEPARATED"
	ShipmentStatusConferred ShipmentStatus = "CONFERRED"
	ShipmentStatusShipped   ShipmentStatus = "SHIPPED"
	ShipmentStatusCancelled ShipmentStatus = "CANCELLED"
)

// Shipment is a dispatch note (romaneio de carregamento) for a sales order.
type Shipment struct {
	ID             int64
	Code           int64
	SalesOrderCode *int64
	CarrierCode    *int64
	Status         ShipmentStatus
	TotalVolumes   int
	TotalWeight    float64
	Notes          *string
	ShippedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
	Items          []*ShipmentItem
}

type ShipmentItem struct {
	ID                 int64
	ShipmentID         int64
	Sequence           int
	ItemCode           int64
	SalesOrderItemCode *int64
	WarehouseID        *int64
	Quantity           float64
	ConferredQty       float64
	IsConferred        bool
	Notes              *string
	CreatedAt          time.Time
}
