package purchase_order_uc

import (
	"testing"
	"time"
)

func TestValidateConsultationFilterDefaultsAndValidatesConversion(t *testing.T) {
	f := PurchaseOrderConsultationFilter{}
	if err := validateConsultationFilter(&f); err != nil {
		t.Fatal(err)
	}
	if f.Limit != 100 {
		t.Fatalf("limit=%d want 100", f.Limit)
	}
	if err := validateConsultationFilter(&PurchaseOrderConsultationFilter{Convert: true}); err == nil {
		t.Fatal("expected required conversion fields error")
	}
	d := time.Now()
	f = PurchaseOrderConsultationFilter{Convert: true, TargetCurrency: "usd", BaseDate: &d, Position: "pending", OrderType: "ocl"}
	if err := validateConsultationFilter(&f); err != nil {
		t.Fatal(err)
	}
	if f.TargetCurrency != "USD" || f.Position != PositionPending || f.OrderType != "OCL" {
		t.Fatalf("not normalized: %+v", f)
	}
}

func TestValidateConsultationFilterRejectsInvalidRangesAndEnums(t *testing.T) {
	a, b := int64(2), int64(1)
	for _, f := range []PurchaseOrderConsultationFilter{{OrderFrom: &a, OrderTo: &b}, {Position: "OTHER"}, {OrderType: "XXX"}, {Limit: 501}, {Offset: -1}} {
		if err := validateConsultationFilter(&f); err == nil {
			t.Fatalf("expected error for %+v", f)
		}
	}
}
