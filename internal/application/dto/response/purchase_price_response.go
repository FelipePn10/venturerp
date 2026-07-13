package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PurchasePriceTableResponse struct {
	ID            int64                            `json:"id"`
	EnterpriseID  int64                            `json:"enterprise_id"`
	Code          int64                            `json:"code"`
	SupplierCode  int64                            `json:"supplier_code"`
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

type PriceAdjustmentResponse struct {
	ID              int64           `json:"id"`
	Sequence        int32           `json:"sequence"`
	Kind            string          `json:"kind"`
	CalculationType string          `json:"calculation_type"`
	Value           decimal.Decimal `json:"value"`
}

type PurchasePriceTableItemResponse struct {
	ID                     int64                     `json:"id"`
	TableID                int64                     `json:"table_id"`
	ItemCode               int64                     `json:"item_code"`
	SupplierCode           *int64                    `json:"supplier_code,omitempty"`
	UOM                    *string                   `json:"uom,omitempty"`
	Price                  decimal.Decimal           `json:"price"`
	MinQty                 decimal.Decimal           `json:"min_qty"`
	UpdateReplacementValue bool                      `json:"update_replacement_value"`
	IsActive               bool                      `json:"is_active"`
	CreatedAt              time.Time                 `json:"created_at"`
	UpdatedAt              time.Time                 `json:"updated_at"`
	Adjustments            []PriceAdjustmentResponse `json:"adjustments,omitempty"`
}

type PurchasePriceItemCandidateResponse struct {
	ItemCode            int64   `json:"item_code"`
	InternalDescription string  `json:"internal_description"`
	SupplierItemCode    *string `json:"supplier_item_code,omitempty"`
	SupplierDescription *string `json:"supplier_description,omitempty"`
	UOM                 *string `json:"uom,omitempty"`
}
type PurchasePriceSourceResponse struct {
	SourceType   string          `json:"source_type"`
	SourceID     int64           `json:"source_id"`
	DocumentCode int64           `json:"document_code"`
	DocumentDate time.Time       `json:"document_date"`
	SupplierCode int64           `json:"supplier_code"`
	ItemCode     int64           `json:"item_code"`
	UOM          string          `json:"uom"`
	UnitPrice    decimal.Decimal `json:"unit_price"`
}
