package request

import "github.com/shopspring/decimal"

type UpsertPurchaseToleranceDTO struct {
	ID             int64            `json:"id,omitempty"`
	ToleranceType  string           `json:"tolerance_type"`
	AppliesTo      string           `json:"applies_to"`
	IntervalMin    decimal.Decimal  `json:"interval_min"`
	IntervalMax    *decimal.Decimal `json:"interval_max,omitempty"`
	ToleranceValue decimal.Decimal  `json:"tolerance_value"`
	ValueType      string           `json:"value_type"`
	SupplierCode   *int64           `json:"supplier_code,omitempty"`
	Action         string           `json:"action"`
	IsActive       *bool            `json:"is_active,omitempty"`
}
type EvaluatePurchaseToleranceDTO struct {
	ToleranceType string          `json:"tolerance_type"`
	AppliesTo     string          `json:"applies_to"`
	SupplierCode  *int64          `json:"supplier_code,omitempty"`
	Expected      decimal.Decimal `json:"expected"`
	Actual        decimal.Decimal `json:"actual"`
}
