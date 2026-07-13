package third_party_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPriceValidationAndPending(t *testing.T) {
	p := &Price{ItemCode: 1, SupplierCode: 2, OperationID: 3, UOM: "kg", ReferenceDate: time.Now(), UnitPrice: decimal.Zero, FreightType: "fixed", FreightValue: decimal.Zero, TaxPercent: decimal.Zero}
	if e := p.Validate(); e != nil {
		t.Fatal(e)
	}
	if p.UOM != "KG" || p.FreightType != "FIXED" {
		t.Fatalf("normalization failed: %+v", p)
	}
	o := ServiceOrder{Quantity: decimal.NewFromInt(10), FulfilledQuantity: decimal.NewFromInt(4)}
	if !o.Pending().Equal(decimal.NewFromInt(6)) {
		t.Fatalf("pending=%s", o.Pending())
	}
}
func TestEvaluateFormulaUsesDecimalArithmetic(t *testing.T) {
	v, e := EvaluateFormula("BASE * QTY + 0.10", map[string]decimal.Decimal{"BASE": decimal.RequireFromString("1.25"), "QTY": decimal.NewFromInt(3)})
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "3.85" {
		t.Fatalf("value=%s", v)
	}
}
