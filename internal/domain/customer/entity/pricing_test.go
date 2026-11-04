package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

var d = decimal.RequireFromString

func TestFormSalesPriceWithMarkupOnly(t *testing.T) {
	got, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:      d("100"),
		MarkupPct:     d("25"),
		DecimalPlaces: 2,
	})
	if err != nil {
		t.Fatalf("FormSalesPrice returned error: %v", err)
	}
	if !got.SuggestedPrice.Equal(d("125")) {
		t.Fatalf("suggested price = %s, want 125.00", got.SuggestedPrice)
	}
	if !got.ContributionMarginValue.Equal(d("25")) {
		t.Fatalf("contribution margin value = %s, want 25.00", got.ContributionMarginValue)
	}
}

func TestFormSalesPriceWithCommercialLoads(t *testing.T) {
	got, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:      d("100"),
		MarginPct:     d("20"),
		TaxesPct:      d("10"),
		CommissionPct: d("5"),
		DecimalPlaces: 2,
	})
	if err != nil {
		t.Fatalf("FormSalesPrice returned error: %v", err)
	}
	if !got.SuggestedPrice.Equal(d("153.85")) {
		t.Fatalf("suggested price = %s, want 153.85", got.SuggestedPrice)
	}
}

// 1,15 × 1,10 = 1,265 exato. Em float o produto era 1,2649999999999999 e o
// arredondamento a centavos dava 1,26; em decimal o meio centavo sobe: 1,27.
func TestFormSalesPriceRoundsHalfCentUp(t *testing.T) {
	got, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:      d("1.15"),
		MarkupPct:     d("10"),
		DecimalPlaces: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.SuggestedPrice.Equal(d("1.27")) {
		t.Fatalf("suggested price = %s, want 1.27", got.SuggestedPrice)
	}
}

func TestFormSalesPriceRejectsInvalidLoad(t *testing.T) {
	_, err := FormSalesPrice(SalesPriceFormationInput{
		BaseCost:  d("100"),
		MarginPct: d("70"),
		TaxesPct:  d("30"),
	})
	if err == nil {
		t.Fatal("expected error for load >= 100")
	}
}
