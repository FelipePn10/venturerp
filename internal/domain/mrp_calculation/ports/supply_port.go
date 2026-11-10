package ports

import (
	"context"
	"github.com/shopspring/decimal"
	"time"
)

// SupplySourceType identifies the origin of a confirmed supply entry.
type SupplySourceType string

const (
	SupplySourcePlannedOrder  SupplySourceType = "PLANNED_ORDER"
	SupplySourcePurchaseOrder SupplySourceType = "PURCHASE_ORDER"
	SupplySourceInTransit     SupplySourceType = "IN_TRANSIT"
	// SupplySourceProductionOrder: OF aberta sem ordem planejada firme.
	SupplySourceProductionOrder SupplySourceType = "PRODUCTION_ORDER"
)

// SupplyEntry represents one unit of firm (approved) supply used in
// time-phased net requirements netting.
// Only entries whose ArrivalDate <= need_date reduce the net requirement.
type SupplyEntry struct {
	ItemCode    int64
	Quantity    decimal.Decimal
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

// OrderAllocation é o empenho de uma OF aberta: material da lista que ainda
// não saiu do estoque. Para o MRP é demanda dependente do componente — como a
// reserva da ordem de produção no MD04 do SAP.
type OrderAllocation struct {
	ProductionOrderID int64
	ParentItemCode    int64
	ParentMask        string
	ParentQuantity    decimal.Decimal
	ItemCode          int64
	Mask              string
	Quantity          decimal.Decimal
	OrderStart        time.Time
	OrderEnd          time.Time
	// Step: etapa do roteiro do pai em que o componente entra (nil = início).
	Step *int16
}

// OrderAllocationsPort é opcional: o adaptador de suprimento que a implementa
// faz o MRP planejar o componente das ordens já abertas.
type OrderAllocationsPort interface {
	ListOpenOrderAllocations(ctx context.Context) ([]OrderAllocation, error)
}

// FirmPlannedWithoutOrder é a ordem planejada de fabricação firme que ainda não
// virou OF: suprimento do item e, portanto, demanda dos componentes.
type FirmPlannedWithoutOrder struct {
	Code     int64
	ItemCode int64
	Mask     string
	Quantity decimal.Decimal
	Start    time.Time
	Need     time.Time
}

// FirmPlannedWithoutOrderPort é opcional no adaptador de suprimento.
type FirmPlannedWithoutOrderPort interface {
	ListFirmPlannedProductionWithoutOrder(ctx context.Context) ([]FirmPlannedWithoutOrder, error)
}
