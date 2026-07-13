package response

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type PurchaseToleranceResponse struct {
	ID             int64            `json:"id"`
	ToleranceType  string           `json:"tolerance_type"`
	AppliesTo      string           `json:"applies_to"`
	IntervalMin    decimal.Decimal  `json:"interval_min"`
	IntervalMax    *decimal.Decimal `json:"interval_max,omitempty"`
	ToleranceValue decimal.Decimal  `json:"tolerance_value"`
	ValueType      string           `json:"value_type"`
	SupplierCode   *int64           `json:"supplier_code,omitempty"`
	Action         string           `json:"action"`
	IsActive       bool             `json:"is_active"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	CreatedBy      uuid.UUID        `json:"created_by"`
}
