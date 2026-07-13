package entity

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestEvaluatePercentAndFixed(t *testing.T) {
	p, err := New(1, ToleranceQuantity, AppliesAll, decimal.Zero, nil, decimal.NewFromInt(5), ValuePercent, nil, ActionBlock, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Evaluate(decimal.NewFromInt(100), decimal.NewFromInt(106)); !got.Exceeded || !got.Allowed.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("unexpected evaluation: %+v", got)
	}
	f, _ := New(1, ToleranceItemPrice, AppliesEntryInvoice, decimal.Zero, nil, decimal.RequireFromString("0.50"), ValueFixed, nil, ActionWarn, uuid.New())
	if got := f.Evaluate(decimal.NewFromInt(10), decimal.RequireFromString("10.40")); got.Exceeded {
		t.Fatalf("fixed tolerance should allow: %+v", got)
	}
}
