package entity

import (
	"time"

	"github.com/google/uuid"
)

// StockLot is the registry of a raw-material lot: its supplier lot, heat number
// (corrida) and quality certificate — the traceability metadata a metallurgy
// shop must keep to answer "which heat went into this part?".
type StockLot struct {
	ID           int64
	ItemCode     int64
	Lot          string
	HeatNumber   *string
	Certificate  *string
	SupplierCode *int64
	ReceivedAt   *time.Time
	Notes        *string
	CreatedAt    time.Time
	CreatedBy    uuid.UUID
}

// StockLotBalance is the on-hand quantity of a single lot in a warehouse.
type StockLotBalance struct {
	ID             int64
	ItemCode       int64
	Mask           string
	WarehouseID    int64
	Lot            string
	Quantity       float64
	LastCost       float64
	LastMovementAt *time.Time
	UpdatedAt      time.Time
}

// LotGenealogy is the full traceability of an item lot, in both directions:
// where a raw-material lot was consumed, and what input lots a produced lot is
// made of.
type LotGenealogy struct {
	ItemCode   int64
	Lot        string
	Registry   *StockLot
	Balances   []*StockLotBalance
	ConsumedIn []LotConsumption // production orders that consumed this lot
	ProducedBy []LotProduction  // production orders that produced this lot
}

// LotConsumption is a production order that consumed the queried lot.
type LotConsumption struct {
	ProductionOrderID int64
	OrderNumber       int64
	ProducedItemCode  int64
	ConsumedQty       float64
}

// LotProduction is a production order that produced the queried lot, together
// with the input lots that went into it.
type LotProduction struct {
	ProductionOrderID int64
	OrderNumber       int64
	ProducedQty       float64
	InputLots         []LotInput
}

// LotInput is a raw-material lot consumed by a production order.
type LotInput struct {
	ItemCode    int64
	Lot         string
	ConsumedQty float64
}
