package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type OverheadAllocation struct {
	Code            int64
	CostCenterCode  int64
	PlanAccountCode *int64
	AccountCode     *string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	AllocationType  string
	BaseCode        *int64
	Targets         []*AllocationTarget
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       uuid.UUID
}

type AllocationTarget struct {
	Code           int64
	OverheadCode   int64
	CostCenterCode int64
	Percentage     decimal.Decimal
	Amount         decimal.Decimal
	CreatedAt      time.Time
}
