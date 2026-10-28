package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestEvaluateCommercialPoliciesAppliesDiscountFreightAndCommission(t *testing.T) {
	itemCode := "1001"
	policies := []*CommercialPolicy{
		{
			ID:           1,
			Code:         10,
			Description:  "Volume",
			Kind:         CommercialPolicyDiscount,
			CalcType:     CommercialPolicyPercent,
			PercentValue: 10,
			MinQuantity:  5,
			Stackable:    true,
			IsActive:     true,
			ItemCode:     &itemCode,
		},
		{
			ID:          2,
			Code:        20,
			Description: "Freight",
			Kind:        CommercialPolicyFreight,
			CalcType:    CommercialPolicyValue,
			FixedValue:  50,
			Stackable:   true,
			IsActive:    true,
		},
		{
			ID:           3,
			Code:         30,
			Description:  "Commission",
			Kind:         CommercialPolicyCommission,
			CalcType:     CommercialPolicyPercent,
			PercentValue: 5,
			Stackable:    true,
			IsActive:     true,
		},
	}

	got, err := EvaluateCommercialPolicies(policies, CommercialPolicyContext{
		GrossValue: decimal.NewFromFloat(1000),
		Quantity:   decimal.NewFromFloat(10),
		ItemCode:   &itemCode,
	})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if !got.DiscountValue.Equal(decimal.NewFromFloat(100)) {
		t.Fatalf("discount = %s, want 100.00", got.DiscountValue)
	}
	if !got.FreightValue.Equal(decimal.NewFromFloat(50)) {
		t.Fatalf("freight = %s, want 50.00", got.FreightValue)
	}
	if !got.CommissionValue.Equal(decimal.NewFromFloat(47.5)) {
		t.Fatalf("commission = %s, want 47.50", got.CommissionValue)
	}
	if !got.NetValue.Equal(decimal.NewFromFloat(950)) {
		t.Fatalf("net = %s, want 950.00", got.NetValue)
	}
}

func TestEvaluateCommercialPoliciesSkipsNonMatchingPolicy(t *testing.T) {
	itemCode := "1001"
	otherItem := "9999"
	got, err := EvaluateCommercialPolicies([]*CommercialPolicy{
		{
			Code:         10,
			Description:  "Wrong item",
			Kind:         CommercialPolicyDiscount,
			CalcType:     CommercialPolicyPercent,
			PercentValue: 10,
			Stackable:    true,
			IsActive:     true,
			ItemCode:     &otherItem,
		},
	}, CommercialPolicyContext{GrossValue: decimal.NewFromFloat(1000), Quantity: decimal.NewFromFloat(1), ItemCode: &itemCode})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if !got.NetValue.Equal(decimal.NewFromFloat(1000)) || len(got.Effects) != 0 {
		t.Fatalf("unexpected application: net %s effects %d", got.NetValue, len(got.Effects))
	}
}

func TestEvaluateCommercialPoliciesHonorsNonStackableKind(t *testing.T) {
	policies := []*CommercialPolicy{
		{
			Code:         10,
			Description:  "First",
			Kind:         CommercialPolicyDiscount,
			CalcType:     CommercialPolicyPercent,
			PercentValue: 10,
			Stackable:    false,
			IsActive:     true,
		},
		{
			Code:         11,
			Description:  "Second",
			Kind:         CommercialPolicyDiscount,
			CalcType:     CommercialPolicyPercent,
			PercentValue: 10,
			Stackable:    true,
			IsActive:     true,
		},
	}
	got, err := EvaluateCommercialPolicies(policies, CommercialPolicyContext{GrossValue: decimal.NewFromFloat(1000), Quantity: decimal.NewFromFloat(1)})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if !got.DiscountValue.Equal(decimal.NewFromFloat(100)) || len(got.Effects) != 1 {
		t.Fatalf("discount %s effects %d, want 100.00 and 1", got.DiscountValue, len(got.Effects))
	}
}

func TestEvaluateCommercialPoliciesUsesPolicyLineWhenPresent(t *testing.T) {
	policies := []*CommercialPolicy{
		{
			Code:         10,
			Description:  "Line discount",
			Kind:         CommercialPolicyDiscount,
			CalcType:     CommercialPolicyPercent,
			PercentValue: 2,
			Stackable:    true,
			IsActive:     true,
			Lines: []*CommercialPolicyLine{
				{
					LineNumber:     1,
					SequenceNumber: 1,
					CalcType:       CommercialPolicyPercent,
					PercentValue:   7,
					IsActive:       true,
				},
			},
		},
	}
	got, err := EvaluateCommercialPolicies(policies, CommercialPolicyContext{GrossValue: decimal.NewFromFloat(1000), Quantity: decimal.NewFromFloat(1)})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if !got.DiscountValue.Equal(decimal.NewFromFloat(70)) {
		t.Fatalf("discount %s, want 70.00", got.DiscountValue)
	}
}

// Descontos encadeados sobre o líquido, centavo a centavo em decimal:
// 100,10 → 10,01 → 9,01 (90,09) → 8,11 (81,08) = 27,13; líquido 72,97.
func TestEvaluateCommercialPoliciesDecimalEncadeado(t *testing.T) {
	var policies []*CommercialPolicy
	for i := int64(1); i <= 3; i++ {
		policies = append(policies, &CommercialPolicy{ID: i, Code: i, Kind: CommercialPolicyDiscount, CalcType: CommercialPolicyPercent, PercentValue: 10, Stackable: true, IsActive: true})
	}
	got, err := EvaluateCommercialPolicies(policies, CommercialPolicyContext{GrossValue: decimal.RequireFromString("100.10"), Quantity: decimal.NewFromInt(1)})
	if err != nil {
		t.Fatal(err)
	}
	if !got.DiscountValue.Equal(decimal.RequireFromString("27.13")) || !got.NetValue.Equal(decimal.RequireFromString("72.97")) {
		t.Fatalf("desconto %s líquido %s", got.DiscountValue, got.NetValue)
	}
}
