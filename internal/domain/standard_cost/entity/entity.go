package entity

import (
	"time"

	"github.com/google/uuid"
)

type ItemStandardCost struct {
	ID           int64
	ItemCode     int64
	Mask         string
	MaterialCost float64
	LaborCost    float64
	OverheadCost float64
	TotalCost    float64
	Currency     string
	CalculatedAt time.Time
	CalculatedBy uuid.UUID
}

type WorkCenterCost struct {
	ID           int64
	WorkCenterID int64
	CostPerHour  float64
	Currency     string
	UpdatedAt    time.Time
	UpdatedBy    uuid.UUID
}

type ItemPurchaseCost struct {
	ID        int64
	ItemCode  int64
	UnitCost  float64
	Currency  string
	UpdatedAt time.Time
	UpdatedBy uuid.UUID
}

type CostRollupLogEntry struct {
	ID           int64
	ItemCode     int64
	Mask         string
	BOMLevel     int
	MaterialCost float64
	LaborCost    float64
	OverheadCost float64
	RunAt        time.Time
}

type RollupResult struct {
	ItemCode     int64
	Mask         string
	MaterialCost float64
	LaborCost    float64
	OverheadCost float64
	TotalCost    float64
	Detail       []CostRollupLogEntry
}
