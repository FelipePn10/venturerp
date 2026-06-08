package entity

import (
	"time"

	"github.com/google/uuid"
)

// Movement type values stored in stock_movements.movement_type.
// These are the canonical strings expected by the stock_balances update logic
// and by the financial reports that filter on movement_type.
const (
	MovementTypeIn          = "IN"
	MovementTypeOut         = "OUT"
	MovementTypeTransferIn  = "TRANSFER_IN"
	MovementTypeTransferOut = "TRANSFER_OUT"
	MovementTypeAdjustment  = "ADJUSTMENT"
)

// Reference type values stored in stock_movements.reference_type, identifying the
// document that originated the movement.
const (
	ReferenceTypePurchaseOrder   = "PURCHASE_ORDER"
	ReferenceTypeProductionOrder = "PRODUCTION_ORDER"
	ReferenceTypeSalesOrder      = "SALES_ORDER"
	ReferenceTypeNFEntry         = "NF_ENTRADA"
	ReferenceTypeNFExit          = "NF_SAIDA"
	ReferenceTypeInventory       = "INVENTORY"
	ReferenceTypeManual          = "MANUAL"
)

// SignedQuantity returns the delta that a movement of the given type applies to
// the on-hand balance: positive for inbound, negative for outbound. ADJUSTMENT
// is assumed to carry an already-signed quantity. Types that do not change the
// on-hand quantity (e.g. reservations) return 0.
func SignedQuantity(movementType string, quantity float64) float64 {
	switch movementType {
	case MovementTypeIn, MovementTypeTransferIn, "ENTRADA":
		return quantity
	case MovementTypeOut, MovementTypeTransferOut, "SAIDA":
		return -quantity
	case MovementTypeAdjustment:
		return quantity
	default:
		return 0
	}
}

type StockMovement struct {
	ID             int64
	ItemCode       int64
	Mask           string
	WarehouseID    int64
	MovementType   string
	Quantity       float64
	UnitPrice      float64
	TotalPrice     float64
	ReferenceType  *string
	ReferenceCode  *int64
	Lot            *string
	SerialNumber   *string
	Batch          *string
	ExpirationDate *time.Time
	Notes          *string
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
}

type StockReservation struct {
	ID                int64
	ItemCode          int64
	Mask              string
	WarehouseID       int64
	Quantity          float64
	ReferenceType     string
	ReferenceCode     int64
	ReferenceItemCode *int64
	ReservationDate   time.Time
	ExpirationDate    *time.Time
	Status            string
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         uuid.UUID
}

type StockBalance struct {
	ID             int64
	ItemCode       int64
	Mask           string
	WarehouseID    int64
	Quantity       float64
	ReservedQty    float64
	AvailableQty   float64
	MinimumStock   float64
	MaximumStock   float64
	SafetyStock    float64
	AvgCost        float64
	LastCost       float64
	TotalCost      float64
	LastMovementAt *time.Time
	UpdatedAt      time.Time
}

type PhysicalInventory struct {
	ID           int64
	Code         int64
	Description  string
	WarehouseID  int64
	StartDate    time.Time
	EndDate      *time.Time
	Status       string
	TotalItems   int
	CountedItems int
	Notes        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
}

type PhysicalInventoryItem struct {
	ID               int64
	InventoryID      int64
	ItemCode         int64
	Mask             string
	WarehouseID      int64
	SystemQty        float64
	CountedQty       *float64
	DifferenceQty    *float64
	UnitCost         *float64
	AdjustmentType   *string
	AdjustmentReason *string
	CountedBy        *uuid.UUID
	CountedAt        *time.Time
	IsAdjusted       bool
	CreatedAt        time.Time
}
