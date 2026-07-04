package entity

import "testing"

func TestFormSalesPriceWithMarkupOnly(t *testing.T) {
	got, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:      100,
		MarkupPct:     25,
		DecimalPlaces: 2,
	})
	if err != nil {
		t.Fatalf("FormSalesPrice returned error: %v", err)
	}
	if got.SuggestedPrice != 125 {
		t.Fatalf("suggested price = %.2f, want 125.00", got.SuggestedPrice)
	}
	if got.ContributionMarginValue != 25 {
		t.Fatalf("contribution margin value = %.2f, want 25.00", got.ContributionMarginValue)
	}
}

func TestFormSalesPriceWithCommercialLoads(t *testing.T) {
	got, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:      100,
		MarginPct:     20,
		TaxesPct:      10,
		CommissionPct: 5,
		DecimalPlaces: 2,
	})
	if err != nil {
		t.Fatalf("FormSalesPrice returned error: %v", err)
	}
	if got.SuggestedPrice != 153.85 {
		t.Fatalf("suggested price = %.2f, want 153.85", got.SuggestedPrice)
	}
}

func TestFormSalesPriceRejectsInvalidLoad(t *testing.T) {
	_, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:  100,
		MarginPct: 70,
		TaxesPct:  30,
	})
	if err == nil {
		t.Fatal("expected error for load >= 100")
	}
}
