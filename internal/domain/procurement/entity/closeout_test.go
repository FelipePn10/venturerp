package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func TestComputeLandedCosts_ByValue(t *testing.T) {
	p := &ImportProcess{
		ExchangeRate:   d(5.0), // 1 USD = 5 BRL
		ApportionBasis: "VALUE",
		Items: []*ImportProcessItem{
			{Quantity: d(10), FobUnitPrice: d(2)}, // FOB local = 10*2*5 = 100
			{Quantity: d(10), FobUnitPrice: d(6)}, // FOB local = 10*6*5 = 300
		},
		Expenses: []*ImportExpense{
			{ExpenseType: "FREIGHT", Amount: d(400), InItemCost: true},
			{ExpenseType: "STORAGE", Amount: d(50), InItemCost: false}, // excluded
		},
	}
	ComputeLandedCosts(p)
	// total basis = 100 + 300 = 400; expenses in cost = 400.
	// item0 share 0.25 -> 100 apportioned; landed total 100+100=200; unit 20.
	// item1 share 0.75 -> 300 apportioned; landed total 300+300=600; unit 60.
	if got := p.Items[0].ApportionedExpenses; !got.Equal(d(100)) {
		t.Errorf("item0 apportioned = %v, want 100", got)
	}
	if got := p.Items[0].LandedUnitCost; !got.Equal(d(20)) {
		t.Errorf("item0 landed unit = %v, want 20", got)
	}
	if got := p.Items[1].LandedUnitCost; !got.Equal(d(60)) {
		t.Errorf("item1 landed unit = %v, want 60", got)
	}
}

func TestComputeLandedCosts_EqualSplitWhenNoBasis(t *testing.T) {
	p := &ImportProcess{
		ExchangeRate:   d(1),
		ApportionBasis: "WEIGHT",
		Items: []*ImportProcessItem{
			{Quantity: d(5), Weight: d(0), FobUnitPrice: d(0)},
			{Quantity: d(5), Weight: d(0), FobUnitPrice: d(0)},
		},
		Expenses: []*ImportExpense{{Amount: d(100), InItemCost: true}},
	}
	ComputeLandedCosts(p)
	if !p.Items[0].ApportionedExpenses.Equal(d(50)) || !p.Items[1].ApportionedExpenses.Equal(d(50)) {
		t.Errorf("expected equal split of 50/50, got %v/%v", p.Items[0].ApportionedExpenses, p.Items[1].ApportionedExpenses)
	}
	if !p.Items[0].LandedUnitCost.Equal(d(10)) { // 50/5
		t.Errorf("landed unit = %v, want 10", p.Items[0].LandedUnitCost)
	}
}

func TestDetectEDILineDivergence(t *testing.T) {
	d1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)

	if got := DetectEDILineDivergence(d(100), d(9.5), &d1, d(100), d(9.5), &d1, d(0), d(0)); got != "" {
		t.Errorf("identical lines should have no divergence, got %q", got)
	}
	if got := DetectEDILineDivergence(d(100), d(9.5), &d1, d(90), d(10.0), &d2, d(0), d(0)); got != "QTY,PRICE,DATE" {
		t.Errorf("all-different got %q, want QTY,PRICE,DATE", got)
	}
	if got := DetectEDILineDivergence(d(100), d(9.5), &d1, d(98), d(9.5), &d1, d(5), d(0)); got != "" {
		t.Errorf("qty within tolerance should pass, got %q", got)
	}
	if got := DetectEDILineDivergence(d(100), d(9.5), nil, d(100), d(9.55), nil, d(0), d(0.1)); got != "" {
		t.Errorf("price within tolerance and no dates should pass, got %q", got)
	}
	if got := DetectEDILineDivergence(d(100), d(9.5), nil, d(100), d(9.7), nil, d(0), d(0.1)); got != "PRICE" {
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

// Três linhas iguais e 100 de despesa: a soma das partes fecha com o total.
func TestComputeLandedCosts_RateioFecha(t *testing.T) {
	p := &ImportProcess{ExchangeRate: d(1), ApportionBasis: "QUANTITY",
		Items:    []*ImportProcessItem{{Quantity: d(1)}, {Quantity: d(1)}, {Quantity: d(1)}},
		Expenses: []*ImportExpense{{Amount: d(100), InItemCost: true}}}
	ComputeLandedCosts(p)
	soma := decimal.Zero
	for _, it := range p.Items {
		soma = soma.Add(it.ApportionedExpenses)
	}
	if !soma.Equal(d(100)) || !p.Items[0].ApportionedExpenses.Equal(d(33.3333)) {
		t.Fatalf("rateio: %v / soma %v", p.Items[0].ApportionedExpenses, soma)
	}
}
