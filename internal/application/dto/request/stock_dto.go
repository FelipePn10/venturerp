package request

type CreateStockMovementDTO struct {
	ItemCode       int64   `json:"item_code"`
	Mask           string  `json:"mask"`
	WarehouseID    int64   `json:"warehouse_id"`
	MovementType   string  `json:"movement_type"`
	Quantity       float64 `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	TotalPrice     float64 `json:"total_price"`
	ReferenceType  *string `json:"reference_type,omitempty"`
	ReferenceCode  *int64  `json:"reference_code,omitempty"`
	Lot            *string `json:"lot,omitempty"`
	SerialNumber   *string `json:"serial_number,omitempty"`
	Batch          *string `json:"batch,omitempty"`
	ExpirationDate *string `json:"expiration_date,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

type CreateReservationDTO struct {
	ItemCode          int64   `json:"item_code"`
	Mask              string  `json:"mask"`
	WarehouseID       int64   `json:"warehouse_id"`
	Quantity          float64 `json:"quantity"`
	ReferenceType     string  `json:"reference_type"`
	ReferenceCode     int64   `json:"reference_code"`
	ReferenceItemCode *int64  `json:"reference_item_code,omitempty"`
	ReservationDate   *string `json:"reservation_date,omitempty"`
	ExpirationDate    *string `json:"expiration_date,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

type CreateInventoryDTO struct {
	Code        int64   `json:"code"`
	Description string  `json:"description"`
	WarehouseID int64   `json:"warehouse_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type CountInventoryItemDTO struct {
	InventoryID      int64    `json:"inventory_id"`
	ItemCode         int64    `json:"item_code"`
	Mask             string   `json:"mask"`
	WarehouseID      int64    `json:"warehouse_id"`
	CountedQty       float64  `json:"counted_qty"`
	UnitCost         *float64 `json:"unit_cost,omitempty"`
	AdjustmentType   *string  `json:"adjustment_type,omitempty"`
	AdjustmentReason *string  `json:"adjustment_reason,omitempty"`
	CountedBy        *string  `json:"counted_by,omitempty"`
}

type AdjustInventoryItemDTO struct {
	InventoryID      int64   `json:"inventory_id"`
	ItemCode         int64   `json:"item_code"`
	Mask             string  `json:"mask"`
	WarehouseID      int64   `json:"warehouse_id"`
	AdjustmentType   string  `json:"adjustment_type"`
	AdjustmentReason *string `json:"adjustment_reason,omitempty"`
}
