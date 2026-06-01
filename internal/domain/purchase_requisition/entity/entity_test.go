package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewPurchaseRequisition(t *testing.T) {
	if _, err := NewPurchaseRequisition(0, 1, uuid.New()); err == nil {
		t.Error("expected error for code 0")
	}
	if _, err := NewPurchaseRequisition(1, 0, uuid.New()); err == nil {
		t.Error("expected error for enterprise 0")
	}
	r, err := NewPurchaseRequisition(1, 10, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != ReqStatusOpen || !r.IsActive {
		t.Errorf("new requisition should be OPEN and active, got %s active=%v", r.Status, r.IsActive)
	}
}

func TestRequisitionItem_Balance(t *testing.T) {
	cases := []struct {
		name                            string
		qty, attended, cancelled, want float64
	}{
		{"nothing attended", 100, 0, 0, 100},
		{"partially attended", 100, 30, 0, 70},
		{"attended + cancelled", 100, 30, 20, 50},
		{"fully attended", 100, 100, 0, 0},
		{"over-attended floors at 0", 100, 90, 20, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it := &PurchaseRequisitionItem{Quantity: tc.qty, AttendedQty: tc.attended, CancelledQty: tc.cancelled}
			if got := it.Balance(); got != tc.want {
				t.Errorf("Balance() = %v, want %v", got, tc.want)
			}
		})
	}
}
