package entity

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

type IndependentDemand struct {
	CodeDemand     int64
	ItemCode       int64
	Mask           *string
	CostCenterCode *int64
	Quantity       decimal.Decimal
	DemandDate     time.Time
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
}
