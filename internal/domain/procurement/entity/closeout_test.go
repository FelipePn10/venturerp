package entity

import (
	"testing"
	"time"
)

func TestComputeLandedCosts_ByValue(t *testing.T) {
	p := &ImportProcess{
		ExchangeRate:   5.0, // 1 USD = 5 BRL
		ApportionBasis: "VALUE",
		Items: []*ImportProcessItem{
			{Quantity: 10, FobUnitPrice: 2}, // FOB local = 10*2*5 = 100
			{Quantity: 10, FobUnitPrice: 6}, // FOB local = 10*6*5 = 300
		},
		Expenses: []*ImportExpense{
			{ExpenseType: "FREIGHT", Amount: 400, InItemCost: true},
			{ExpenseType: "STORAGE", Amount: 50, InItemCost: false}, // excluded
		},
	}
	ComputeLandedCosts(p)
	// total basis = 100 + 300 = 400; expenses in cost = 400.
	// item0 share 0.25 -> 100 apportioned; landed total 100+100=200; unit 20.
	// item1 share 0.75 -> 300 apportioned; landed total 300+300=600; unit 60.
	if got := p.Items[0].ApportionedExpenses; got != 100 {
		t.Errorf("item0 apportioned = %v, want 100", got)
	}
	if got := p.Items[0].LandedUnitCost; got != 20 {
		t.Errorf("item0 landed unit = %v, want 20", got)
	}
	if got := p.Items[1].LandedUnitCost; got != 60 {
		t.Errorf("item1 landed unit = %v, want 60", got)
	}
}

func TestComputeLandedCosts_EqualSplitWhenNoBasis(t *testing.T) {
	p := &ImportProcess{
		ExchangeRate:   1,
		ApportionBasis: "WEIGHT",
		Items: []*ImportProcessItem{
			{Quantity: 5, Weight: 0, FobUnitPrice: 0},
			{Quantity: 5, Weight: 0, FobUnitPrice: 0},
		},
		Expenses: []*ImportExpense{{Amount: 100, InItemCost: true}},
	}
	ComputeLandedCosts(p)
	if p.Items[0].ApportionedExpenses != 50 || p.Items[1].ApportionedExpenses != 50 {
		t.Errorf("expected equal split of 50/50, got %v/%v", p.Items[0].ApportionedExpenses, p.Items[1].ApportionedExpenses)
	}
	if p.Items[0].LandedUnitCost != 10 { // 50/5
		t.Errorf("landed unit = %v, want 10", p.Items[0].LandedUnitCost)
	}
}

func TestDetectEDILineDivergence(t *testing.T) {
	d1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)

	if got := DetectEDILineDivergence(100, 9.5, &d1, 100, 9.5, &d1, 0, 0); got != "" {
		t.Errorf("identical lines should have no divergence, got %q", got)
	}
	if got := DetectEDILineDivergence(100, 9.5, &d1, 90, 10.0, &d2, 0, 0); got != "QTY,PRICE,DATE" {
		t.Errorf("all-different got %q, want QTY,PRICE,DATE", got)
	}
	if got := DetectEDILineDivergence(100, 9.5, &d1, 98, 9.5, &d1, 5, 0); got != "" {
		t.Errorf("qty within tolerance should pass, got %q", got)
	}
	if got := DetectEDILineDivergence(100, 9.5, nil, 100, 9.55, nil, 0, 0.1); got != "" {
		t.Errorf("price within tolerance and no dates should pass, got %q", got)
	}
	if got := DetectEDILineDivergence(100, 9.5, nil, 100, 9.7, nil, 0, 0.1); got != "PRICE" {
		t.Errorf("price beyond tolerance should flag PRICE, got %q", got)
	}
}

func TestHomologationStatusForIQF(t *testing.T) {
	cases := []struct {
		iqf  float64
		want string
	}{
		{95, "HOMOLOGATED"},
		{80, "HOMOLOGATED"},
		{79.99, "CONDITIONAL"},
		{60, "CONDITIONAL"},
		{59, "REJECTED"},
	}
	for _, c := range cases {
		if got := HomologationStatusForIQF(c.iqf, 80, 60); got != c.want {
			t.Errorf("HomologationStatusForIQF(%v)=%q want %q", c.iqf, got, c.want)
		}
	}
}
