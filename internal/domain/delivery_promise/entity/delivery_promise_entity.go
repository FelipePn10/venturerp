package entity

import (
	"time"

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
	Quantity       float64
	UnitPrice      float64
	Source         string
	ReferenceCode  *int64
}

type TankOccupationDay struct {
	TankCode        int64
	Date            time.Time
	Capacity        float64
	Allocated       float64
	Free            float64
	OccupationPct   float64
	Quantity        float64
	ForecastRevenue float64
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
	RequestedQty   float64
	ReservedQty    float64
	AllocationDate time.Time
	ExpiresAt      time.Time
	Status         TankReservationStatus
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
}
