package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreatePurchasePriceTableDTO struct {
	SupplierCode  int64     `json:"supplier_code"`
	Description   string    `json:"description"`
	CurrencyCode  string    `json:"currency_code,omitempty"`
	ValidityStart *string   `json:"validity_start,omitempty"`
	ValidityEnd   *string   `json:"validity_end,omitempty"`
	CreatedBy     uuid.UUID `json:"created_by,omitempty"`
}

type UpdatePurchasePriceTableDTO struct {
	Code          int64   `json:"code"`
	SupplierCode  int64   `json:"supplier_code"`
	Description   string  `json:"description"`
	CurrencyCode  string  `json:"currency_code,omitempty"`
	ValidityStart *string `json:"validity_start,omitempty"`
	ValidityEnd   *string `json:"validity_end,omitempty"`
	IsActive      bool    `json:"is_active"`
}

type PriceAdjustmentDTO struct {
	Sequence        int32           `json:"sequence"`
	Kind            string          `json:"kind"`
	CalculationType string          `json:"calculation_type"`
	Value           decimal.Decimal `json:"value"`
}

type AddPurchasePriceItemDTO struct {
	TableCode              int64                `json:"table_code"`
	ItemCode               int64                `json:"item_code"`
	SupplierCode           *int64               `json:"supplier_code,omitempty"`
	UOM                    *string              `json:"uom,omitempty"`
	Price                  decimal.Decimal      `json:"price"`
	MinQty                 decimal.Decimal      `json:"min_qty"`
	UpdateReplacementValue bool                 `json:"update_replacement_value"`
	Adjustments            []PriceAdjustmentDTO `json:"adjustments,omitempty"`
}

type CopyPriceAdjustmentsDTO struct {
	SourceItemID int64  `json:"source_item_id"`
	TargetItemID int64  `json:"target_item_id"`
	Mode         string `json:"mode"`
}

type ApplyPurchasePriceSourcesDTO struct {
	TableCode  int64 `json:"table_code"`
	Overwrite  bool  `json:"overwrite"`
	Selections []struct {
		SourceType string `json:"source_type"`
		SourceID   int64  `json:"source_id"`
	} `json:"selections"`
}
