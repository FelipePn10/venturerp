package entity

import "testing"

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
		GrossValue: 1000,
		Quantity:   10,
		ItemCode:   &itemCode,
	})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if got.DiscountValue != 100 {
		t.Fatalf("discount = %.2f, want 100.00", got.DiscountValue)
	}
	if got.FreightValue != 50 {
		t.Fatalf("freight = %.2f, want 50.00", got.FreightValue)
	}
	if got.CommissionValue != 47.5 {
		t.Fatalf("commission = %.2f, want 47.50", got.CommissionValue)
	}
	if got.NetValue != 950 {
		t.Fatalf("net = %.2f, want 950.00", got.NetValue)
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
	}, CommercialPolicyContext{GrossValue: 1000, Quantity: 1, ItemCode: &itemCode})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if got.NetValue != 1000 || len(got.Effects) != 0 {
		t.Fatalf("unexpected application: net %.2f effects %d", got.NetValue, len(got.Effects))
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
	got, err := EvaluateCommercialPolicies(policies, CommercialPolicyContext{GrossValue: 1000, Quantity: 1})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if got.DiscountValue != 100 || len(got.Effects) != 1 {
		t.Fatalf("discount %.2f effects %d, want 100.00 and 1", got.DiscountValue, len(got.Effects))
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
	got, err := EvaluateCommercialPolicies(policies, CommercialPolicyContext{GrossValue: 1000, Quantity: 1})
	if err != nil {
		t.Fatalf("EvaluateCommercialPolicies returned error: %v", err)
	}
	if got.DiscountValue != 70 {
		t.Fatalf("discount %.2f, want 70.00", got.DiscountValue)
	}
}
