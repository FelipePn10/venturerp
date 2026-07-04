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
	CostPerHour  float64 // legacy blended rate (kept as machine-rate fallback)
	// Enterprise+ split: machine occupancy rate vs. direct-labor rate per hour.
	MachineCostPerHour float64
	LaborCostPerHour   float64
	Currency           string
	UpdatedAt          time.Time
	UpdatedBy          uuid.UUID
}

// MachineRate returns the effective machine hourly rate (falls back to the blended rate).
func (w *WorkCenterCost) MachineRate() float64 {
	if w.MachineCostPerHour > 0 {
		return w.MachineCostPerHour
	}
	return w.CostPerHour
}

// LaborRate returns the effective labor hourly rate.
func (w *WorkCenterCost) LaborRate() float64 { return w.LaborCostPerHour }

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
