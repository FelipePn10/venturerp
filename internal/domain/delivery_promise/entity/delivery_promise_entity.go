package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type TankReservationStatus string

const (
	TankReservationActive    TankReservationStatus = "ACTIVE"
	TankReservationCancelled TankReservationStatus = "CANCELLED"
	TankReservationExpired   TankReservationStatus = "EXPIRED"
)

type TankAllocation struct {
	TankCode       int64
	ItemCode       int64
	Mask           string
	AllocationDate time.Time
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	Source         string
	ReferenceCode  *int64
}

type TankOccupationDay struct {
	TankCode        int64
	Date            time.Time
	Capacity        decimal.Decimal
	Allocated       decimal.Decimal
	Free            decimal.Decimal
	OccupationPct   decimal.Decimal
	Quantity        decimal.Decimal
	ForecastRevenue decimal.Decimal
	Allocations     []TankAllocation
	Warnings        []string
}

type TankReservation struct {
	ID             int64
	Code           int64
	CustomerCode   *int64
	ItemCode       int64
	Mask           string
	TankCode       int64
	RequestedQty   decimal.Decimal
	ReservedQty    decimal.Decimal
	AllocationDate time.Time
	ExpiresAt      time.Time
	Status         TankReservationStatus
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
}
