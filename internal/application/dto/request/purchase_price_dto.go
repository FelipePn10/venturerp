package request

import "github.com/google/uuid"

type CreatePurchasePriceTableDTO struct {
	Description   string    `json:"description"`
	CurrencyCode  string    `json:"currency_code,omitempty"`
	ValidityStart *string   `json:"validity_start,omitempty"`
	ValidityEnd   *string   `json:"validity_end,omitempty"`
	CreatedBy     uuid.UUID `json:"created_by"`
}

type UpdatePurchasePriceTableDTO struct {
	Code          int64   `json:"code"`
	Description   string  `json:"description"`
	CurrencyCode  string  `json:"currency_code,omitempty"`
	ValidityStart *string `json:"validity_start,omitempty"`
	ValidityEnd   *string `json:"validity_end,omitempty"`
	IsActive      bool    `json:"is_active"`
}

type AddPurchasePriceItemDTO struct {
	TableCode    int64   `json:"table_code"`
	ItemCode     int64   `json:"item_code"`
	SupplierCode *int64  `json:"supplier_code,omitempty"`
	UOM          *string `json:"uom,omitempty"`
	Price        float64 `json:"price"`
	MinQty       float64 `json:"min_qty"`
}
