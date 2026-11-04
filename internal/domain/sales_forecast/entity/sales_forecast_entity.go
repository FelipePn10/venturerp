package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type SalesForecast struct {
	ID        int64
	ItemCode  int64
	Mask      *string
	Week      int
	Year      int
	Quantity  decimal.Decimal
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SalesForecastBlock struct {
	ID        int64
	StartDate time.Time
	EndDate   time.Time
	Reason    *string
	CreatedAt time.Time
	CreatedBy uuid.UUID
}

type AppropriationTable struct {
	ID           int64
	Description  string
	MondayPct    decimal.Decimal
	TuesdayPct   decimal.Decimal
	WednesdayPct decimal.Decimal
	ThursdayPct  decimal.Decimal
	FridayPct    decimal.Decimal
	SaturdayPct  decimal.Decimal
	SundayPct    decimal.Decimal
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
}

type HistoricalDemand struct {
	ItemCode    int64
	Mask        *string
	PeriodMonth time.Time
	Quantity    decimal.Decimal
}
