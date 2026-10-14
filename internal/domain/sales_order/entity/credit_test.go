package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dec(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func TestEvaluateCredit_WithinLimitApproved(t *testing.T) {
	d := EvaluateCredit(dec(10000), dec(3000), dec(2000), dec(4000), false)
	if !d.Approved {
		t.Fatalf("expected approved, got reason %q", d.Reason)
	}
	if !d.Available.Equal(dec(5000)) {
		t.Fatalf("available = %v, want 5000", d.Available)
	}
}

func TestEvaluateCredit_ExceededBlocked(t *testing.T) {
	d := EvaluateCredit(dec(10000), dec(5000), dec(3000), dec(4000), false)
	if d.Approved {
		t.Fatalf("expected not approved (5000+3000+4000 > 10000)")
	}
	if d.Reason == "" {
		t.Fatalf("expected a reason when exceeded")
	}
}

func TestEvaluateCredit_NoLimitAlwaysApproved(t *testing.T) {
	d := EvaluateCredit(dec(0), dec(999999), dec(0), dec(5000), false)
	if !d.Approved {
		t.Fatalf("expected approved when no limit configured")
	}
	if d.LimitApplies {
		t.Fatalf("expected LimitApplies=false when limit is 0")
	}
}

func TestEvaluateCredit_BlockedCustomerNeverApproved(t *testing.T) {
	d := EvaluateCredit(dec(0), dec(0), dec(0), dec(1), true)
	if d.Approved {
		t.Fatalf("blocked customer must not be approved")
	}
}

func TestEvaluateCredit_ExactlyAtLimitApproved(t *testing.T) {
	d := EvaluateCredit(dec(10000), dec(4000), dec(2000), dec(4000), false)
	if !d.Approved {
		t.Fatalf("expected approved when exactly at limit")
	}
}

// Em float, 0,10 + 0,20 = 0,30000000000000004 e o pedido no limite exato era bloqueado.
func TestEvaluateCredit_CentavosSemErroDeArredondamento(t *testing.T) {
	d := EvaluateCredit(decimal.RequireFromString("0.30"), decimal.RequireFromString("0.10"),
		decimal.RequireFromString("0.20"), decimal.Zero, false)
	if !d.Approved {
		t.Fatalf("exposição igual ao limite deveria passar: %q", d.Reason)
	}
}
