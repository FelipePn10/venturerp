package ports

import (
	"context"
	"time"
)

// SupplySourceType identifies the origin of a confirmed supply entry.
type SupplySourceType string

const (
	SupplySourcePlannedOrder  SupplySourceType = "PLANNED_ORDER"
	SupplySourcePurchaseOrder SupplySourceType = "PURCHASE_ORDER"
	SupplySourceInTransit     SupplySourceType = "IN_TRANSIT"
)

// SupplyEntry represents one unit of firm (approved) supply used in
// time-phased net requirements netting.
// Only entries whose ArrivalDate <= need_date reduce the net requirement.
type SupplyEntry struct {
	ItemCode    int64
	Quantity    float64
	ArrivalDate time.Time        // expected date the supply will be available
	SourceType  SupplySourceType // what kind of order generated this supply
	SourceCode  int64            // PK of the originating order record
}

// PlannedOrderSupplyPort is the outbound port the MRP engine calls to query
// firm supply. The MRP domain defines this interface; the planned_order module
// implements it when created. Pass nil to NewMRPService to disable netting
// (pre-planned_order behaviour is preserved).
type PlannedOrderSupplyPort interface {
	// ListFirmSupplyForItems returns all firm (approved) supply records for
	// the given item codes, grouped by item_code.
	// Only approved / firm orders must be returned — draft suggestions must not.
	ListFirmSupplyForItems(ctx context.Context, itemCodes []int64) (map[int64][]SupplyEntry, error)
}
