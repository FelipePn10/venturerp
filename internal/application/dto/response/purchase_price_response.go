package response

import (
	"time"

	"github.com/google/uuid"
)

// PurchasePriceTableResponse is the API representation of a purchase price table.
type PurchasePriceTableResponse struct {
	ID            int64                            `json:"id"`
	Code          int64                            `json:"code"`
	Description   string                           `json:"description"`
	CurrencyCode  string                           `json:"currency_code"`
	ValidityStart *time.Time                       `json:"validity_start,omitempty"`
	ValidityEnd   *time.Time                       `json:"validity_end,omitempty"`
	IsActive      bool                             `json:"is_active"`
	CreatedAt     time.Time                        `json:"created_at"`
	CreatedBy     uuid.UUID                        `json:"created_by"`
	UpdatedAt     time.Time                        `json:"updated_at"`
	Items         []PurchasePriceTableItemResponse `json:"items,omitempty"`
}

// PurchasePriceTableItemResponse is the API representation of a price table line.
type PurchasePriceTableItemResponse struct {
	ID           int64     `json:"id"`
	TableID      int64     `json:"table_id"`
	ItemCode     int64     `json:"item_code"`
	SupplierCode *int64    `json:"supplier_code,omitempty"`
	UOM          *string   `json:"uom,omitempty"`
	Price        float64   `json:"price"`
	MinQty       float64   `json:"min_qty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}
