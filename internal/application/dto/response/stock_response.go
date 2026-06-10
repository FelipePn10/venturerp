package response

import (
	"time"

	"github.com/google/uuid"
)

// StockMovementResponse is the API representation of a stock movement.
type StockMovementResponse struct {
	ID             int64      `json:"id"`
	ItemCode       int64      `json:"item_code"`
	Mask           string     `json:"mask"`
	WarehouseID    int64      `json:"warehouse_id"`
	MovementType   string     `json:"movement_type"`
	Quantity       float64    `json:"quantity"`
	UnitPrice      float64    `json:"unit_price"`
	TotalPrice     float64    `json:"total_price"`
	ReferenceType  *string    `json:"reference_type,omitempty"`
	ReferenceCode  *int64     `json:"reference_code,omitempty"`
	Lot            *string    `json:"lot,omitempty"`
	SerialNumber   *string    `json:"serial_number,omitempty"`
	Batch          *string    `json:"batch,omitempty"`
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      uuid.UUID  `json:"created_by"`
}

// StockReservationResponse is the API representation of a stock reservation.
type StockReservationResponse struct {
	ID                int64      `json:"id"`
	ItemCode          int64      `json:"item_code"`
	Mask              string     `json:"mask"`
	WarehouseID       int64      `json:"warehouse_id"`
	Quantity          float64    `json:"quantity"`
	ReferenceType     string     `json:"reference_type"`
	ReferenceCode     int64      `json:"reference_code"`
	ReferenceItemCode *int64     `json:"reference_item_code,omitempty"`
	ReservationDate   time.Time  `json:"reservation_date"`
	ExpirationDate    *time.Time `json:"expiration_date,omitempty"`
	Status            string     `json:"status"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CreatedBy         uuid.UUID  `json:"created_by"`
}

// StockBalanceResponse is the API representation of a stock balance.
type StockBalanceResponse struct {
	ID             int64      `json:"id"`
	ItemCode       int64      `json:"item_code"`
	Mask           string     `json:"mask"`
	WarehouseID    int64      `json:"warehouse_id"`
	Quantity       float64    `json:"quantity"`
	ReservedQty    float64    `json:"reserved_qty"`
	AvailableQty   float64    `json:"available_qty"`
	MinimumStock   float64    `json:"minimum_stock"`
	MaximumStock   float64    `json:"maximum_stock"`
	SafetyStock    float64    `json:"safety_stock"`
	AvgCost        float64    `json:"avg_cost"`
	LastCost       float64    `json:"last_cost"`
	TotalCost      float64    `json:"total_cost"`
	LastMovementAt *time.Time `json:"last_movement_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PhysicalInventoryResponse is the API representation of a physical inventory.
type PhysicalInventoryResponse struct {
	ID           int64      `json:"id"`
	Code         int64      `json:"code"`
	Description  string     `json:"description"`
	WarehouseID  int64      `json:"warehouse_id"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Status       string     `json:"status"`
	TotalItems   int        `json:"total_items"`
	CountedItems int        `json:"counted_items"`
	Notes        *string    `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedBy    uuid.UUID  `json:"created_by"`
}

// PhysicalInventoryItemResponse is the API representation of an inventory line.
type PhysicalInventoryItemResponse struct {
	ID               int64      `json:"id"`
	InventoryID      int64      `json:"inventory_id"`
	ItemCode         int64      `json:"item_code"`
	Mask             string     `json:"mask"`
	WarehouseID      int64      `json:"warehouse_id"`
	SystemQty        float64    `json:"system_qty"`
	CountedQty       *float64   `json:"counted_qty,omitempty"`
	DifferenceQty    *float64   `json:"difference_qty,omitempty"`
	UnitCost         *float64   `json:"unit_cost,omitempty"`
	AdjustmentType   *string    `json:"adjustment_type,omitempty"`
	AdjustmentReason *string    `json:"adjustment_reason,omitempty"`
	CountedBy        *uuid.UUID `json:"counted_by,omitempty"`
	CountedAt        *time.Time `json:"counted_at,omitempty"`
	IsAdjusted       bool       `json:"is_adjusted"`
	CreatedAt        time.Time  `json:"created_at"`
}
