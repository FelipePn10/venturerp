package entity

import (
	"time"

	"github.com/google/uuid"
)

type SalesForecast struct {
	ID        int64
	ItemCode  int64
	Mask      *string
	Week      int
	Year      int
	Quantity  float64
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
	MondayPct    float64
	TuesdayPct   float64
	WednesdayPct float64
	ThursdayPct  float64
	FridayPct    float64
	SaturdayPct  float64
	SundayPct    float64
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
}

type HistoricalDemand struct {
	ItemCode    int64
	Mask        *string
	PeriodMonth time.Time
	Quantity    float64
}
