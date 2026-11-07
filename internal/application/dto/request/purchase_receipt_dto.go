package request

import "github.com/shopspring/decimal"

// ReceivePurchaseOrderDTO registers the physical receipt of purchase order
// lines and posts inbound stock movements for the received quantities.
type ReceivePurchaseOrderDTO struct {
	PurchaseOrderCode int64                         `json:"-"`
	Items             []ReceivePurchaseOrderItemDTO `json:"items"`
	Notes             *string                       `json:"notes,omitempty"`
}

type ReceivePurchaseOrderItemDTO struct {
	PurchaseOrderItemCode int64           `json:"purchase_order_item_code"`
	Quantity              decimal.Decimal `json:"quantity"`
	WarehouseID           int64           `json:"warehouse_id"`
	Lot                   *string         `json:"lot,omitempty"`
	SerialNumber          *string         `json:"serial_number,omitempty"`
	Batch                 *string         `json:"batch,omitempty"`
	ExpirationDate        *string         `json:"expiration_date,omitempty"`
	// HeatNumber (corrida) e Certificate: exigidos pelo cadastro do item.
	HeatNumber  *string `json:"heat_number,omitempty"`
	Certificate *string `json:"certificate,omitempty"`
	// SecondaryQuantity: o peso real da balança, em item com segunda unidade.
	SecondaryQuantity *decimal.Decimal `json:"secondary_quantity,omitempty"`
	// ChecklistConfirmed: quem recebe confirma o check list do item.
	ChecklistConfirmed bool    `json:"checklist_confirmed,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}
