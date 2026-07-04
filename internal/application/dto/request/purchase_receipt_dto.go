package request

// ReceivePurchaseOrderDTO registers the physical receipt of purchase order
// lines and posts inbound stock movements for the received quantities.
type ReceivePurchaseOrderDTO struct {
	PurchaseOrderCode int64                         `json:"-"`
	Items             []ReceivePurchaseOrderItemDTO `json:"items"`
	Notes             *string                       `json:"notes,omitempty"`
}

type ReceivePurchaseOrderItemDTO struct {
	PurchaseOrderItemCode int64   `json:"purchase_order_item_code"`
	Quantity              float64 `json:"quantity"`
	WarehouseID           int64   `json:"warehouse_id"`
	Lot                   *string `json:"lot,omitempty"`
	SerialNumber          *string `json:"serial_number,omitempty"`
	Batch                 *string `json:"batch,omitempty"`
	ExpirationDate        *string `json:"expiration_date,omitempty"`
	Notes                 *string `json:"notes,omitempty"`
}
