package entity

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

const (
	ToleranceQuantity      = "QUANTITY"
	ToleranceItemPrice     = "ITEM_PRICE"
	ToleranceProductsTotal = "PRODUCTS_TOTAL"
	AppliesEntryInvoice    = "ENTRY_INVOICE"
	AppliesReceivingNotice = "RECEIVING_NOTICE"
	AppliesAll             = "ALL"
	ValuePercent           = "PERCENT"
	ValueFixed             = "FIXED"
	ActionBlock            = "BLOCK"
	ActionWarn             = "WARN"
)

type Tolerance struct {
	ID             int64
	EnterpriseID   int64
	ToleranceType  string
	AppliesTo      string
	IntervalMin    decimal.Decimal
	IntervalMax    *decimal.Decimal
	ToleranceValue decimal.Decimal
	ValueType      string
	SupplierCode   *int64
	Action         string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID
}

func New(e int64, t, a string, min decimal.Decimal, max *decimal.Decimal, value decimal.Decimal, valueType string, supplier *int64, action string, by uuid.UUID) (*Tolerance, error) {
	x := &Tolerance{EnterpriseID: e, ToleranceType: strings.ToUpper(t), AppliesTo: strings.ToUpper(a), IntervalMin: min, IntervalMax: max, ToleranceValue: value, ValueType: strings.ToUpper(valueType), SupplierCode: supplier, Action: strings.ToUpper(action), IsActive: true, CreatedBy: by}
	if err := x.Validate(); err != nil {
		return nil, err
	}
	return x, nil
}
func (x *Tolerance) Validate() error {
	if x.EnterpriseID <= 0 || !oneOf(x.ToleranceType, ToleranceQuantity, ToleranceItemPrice, ToleranceProductsTotal) || !oneOf(x.AppliesTo, AppliesEntryInvoice, AppliesReceivingNotice, AppliesAll) || !oneOf(x.ValueType, ValuePercent, ValueFixed) || !oneOf(x.Action, ActionBlock, ActionWarn) || x.IntervalMin.IsNegative() || x.ToleranceValue.IsNegative() || (x.IntervalMax != nil && x.IntervalMax.LessThan(x.IntervalMin)) {
		return fmt.Errorf("invalid purchase tolerance")
	}
	return nil
}
func oneOf(v string, x ...string) bool {
	for _, a := range x {
		if v == a {
			return true
		}
	}
	return false
}

type Evaluation struct {
	Matched     bool            `json:"matched"`
	Exceeded    bool            `json:"exceeded"`
	Action      string          `json:"action,omitempty"`
	Expected    decimal.Decimal `json:"expected"`
	Actual      decimal.Decimal `json:"actual"`
	Deviation   decimal.Decimal `json:"deviation"`
	Allowed     decimal.Decimal `json:"allowed"`
	ToleranceID int64           `json:"tolerance_id,omitempty"`
}

func (x *Tolerance) Evaluate(expected, actual decimal.Decimal) Evaluation {
	d := actual.Sub(expected).Abs()
	allowed := x.ToleranceValue
	if x.ValueType == ValuePercent {
		allowed = expected.Abs().Mul(x.ToleranceValue).Div(decimal.NewFromInt(100))
	}
	return Evaluation{Matched: true, Exceeded: d.GreaterThan(allowed), Action: x.Action, Expected: expected, Actual: actual, Deviation: d, Allowed: allowed, ToleranceID: x.ID}
}
