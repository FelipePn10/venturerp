package entity

import (
	"time"

	"github.com/google/uuid"
)

// ConsumptionMode decides how raw-material lots are chosen when a plan is firmed.
//
//   - AUTOMATIC: the system picks lots by FIFO (oldest received first).
//   - MANUAL:    the operator assigns the lot on each stock piece.
//
// The company sets a default in CuttingSettings; a plan may override it. This is
// the "os dois, ou a empresa decide" requirement.
type ConsumptionMode string

const (
	ConsumptionAutomatic ConsumptionMode = "AUTOMATIC"
	ConsumptionManual    ConsumptionMode = "MANUAL"
)

// RemnantStatus is the lifecycle of a reusable offcut.
type RemnantStatus string

const (
	RemnantAvailable RemnantStatus = "AVAILABLE"
	RemnantReserved  RemnantStatus = "RESERVED"
	RemnantConsumed  RemnantStatus = "CONSUMED"
)

// Consumption source discriminator for traceability records.
const (
	ConsumptionSourceLot     = "LOT"
	ConsumptionSourceRemnant = "REMNANT"
)

// StockRemnant is a reusable offcut: a unique physical piece with its own
// geometry, kept in a dedicated inventory (not fungible stock). It inherits the
// heat number and certificate of the material it was cut from, so traceability
// survives re-cutting.
type StockRemnant struct {
	ID             int64
	ItemCode       int64
	WarehouseID    int64
	LengthMM       float64
	WidthMM        float64 // 2D remnant geometry
	HeightMM       float64 // 2D remnant geometry
	Lot            *string
	HeatNumber     *string
	Certificate    *string
	Status         RemnantStatus
	UnitCost       float64
	OriginPlanID   *int64
	ConsumedPlanID *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
}

// CuttingPlanConsumption records one stock draw made when a plan was firmed —
// the audit trail of what each plan consumed (a lot or a remnant) and at what
// cost.
type CuttingPlanConsumption struct {
	ID          int64
	PlanID      int64
	ItemCode    int64
	SourceType  string // ConsumptionSourceLot | ConsumptionSourceRemnant
	Lot         *string
	RemnantID   *int64
	Quantity    float64 // pieces consumed
	LengthMM    float64 // per-piece length
	UnitCost    float64
	TotalCost   float64
	WarehouseID int64
	MovementID  *int64 // stock_movements.id when a baixa was posted
	CreatedAt   time.Time
}

// LotAvailability is an on-hand lot of the material in a warehouse, used to pick
// lots FIFO when consumption is automatic.
type LotAvailability struct {
	Lot         string
	Quantity    float64
	LastCost    float64
	HeatNumber  *string
	Certificate *string
	ReceivedAt  *time.Time
}

// CuttingPlanOrderCost is the share of a firmed plan's material cost allocated to
// one source order, proportional to that order's demand. Lets a plan that
// aggregates several OPs still cost back to each one.
type CuttingPlanOrderCost struct {
	ID            int64
	PlanID        int64
	OrderRef      string
	DemandMeasure float64 // length (1D) or area (2D)
	AllocatedCost float64
	CreatedAt     time.Time
}

// CuttingSettings is the company-level default for cutting plans (singleton).
type CuttingSettings struct {
	DefaultConsumptionMode ConsumptionMode
	DefaultMinRemnantMM    float64
	DefaultWarehouseID     *int64
	UpdatedAt              time.Time
}

// EffectiveConsumptionMode resolves the plan's mode against the company default.
func (p *CuttingPlan) EffectiveConsumptionMode(settings *CuttingSettings) ConsumptionMode {
	if p.LotConsumptionMode != nil && *p.LotConsumptionMode != "" {
		return *p.LotConsumptionMode
	}
	if settings != nil && settings.DefaultConsumptionMode != "" {
		return settings.DefaultConsumptionMode
	}
	return ConsumptionAutomatic
}
