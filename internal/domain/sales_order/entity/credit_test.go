package entity

import "testing"

func TestEvaluateCredit_WithinLimitApproved(t *testing.T) {
	d := EvaluateCredit(10000, 3000, 2000, 4000, false)
	if !d.Approved {
		t.Fatalf("expected approved, got reason %q", d.Reason)
	}
	if d.Available != 5000 {
		t.Fatalf("available = %v, want 5000", d.Available)
	}
}

func TestEvaluateCredit_ExceededBlocked(t *testing.T) {
	d := EvaluateCredit(10000, 5000, 3000, 4000, false)
	if d.Approved {
		t.Fatalf("expected not approved (5000+3000+4000 > 10000)")
	}
	if d.Reason == "" {
		t.Fatalf("expected a reason when exceeded")
	}
}

func TestEvaluateCredit_NoLimitAlwaysApproved(t *testing.T) {
	d := EvaluateCredit(0, 999999, 0, 5000, false)
	if !d.Approved {
		t.Fatalf("expected approved when no limit configured")
	}
	if d.LimitApplies {
		t.Fatalf("expected LimitApplies=false when limit is 0")
	}
}

func TestEvaluateCredit_BlockedCustomerNeverApproved(t *testing.T) {
	d := EvaluateCredit(0, 0, 0, 1, true)
	if d.Approved {
		t.Fatalf("blocked customer must not be approved")
	}
}

func TestEvaluateCredit_ExactlyAtLimitApproved(t *testing.T) {
	d := EvaluateCredit(10000, 4000, 2000, 4000, false)
	if !d.Approved {
		t.Fatalf("expected approved when exactly at limit")
	}
}
