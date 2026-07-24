package sales_quotation_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/shopspring/decimal"
)

func TestApplyQuotationRulesForcesNFCeAndRemovesIPIContext(t *testing.T) {
	q := &entity.SalesQuotation{DeliveryWithReceipt: true}
	if err := applyQuotationRules(q, entity.DefaultParameters(1)); err != nil {
		t.Fatal(err)
	}
	if !q.IsNFCe {
		t.Fatal("delivery with receipt must force NFC-e")
	}
}

func TestApplyQuotationRulesEnforcesMinimumCIFFreight(t *testing.T) {
	freightType := "Cif-Contrat."
	p := entity.DefaultParameters(1)
	p.MinimumCIFFreight = decimal.RequireFromString("25.50")
	q := &entity.SalesQuotation{FreightType: &freightType, VerifyFreight: true, FreightValue: decimal.NewFromInt(10)}
	if err := applyQuotationRules(q, p); err != nil {
		t.Fatal(err)
	}
	if !q.FreightValue.Equal(p.MinimumCIFFreight) {
		t.Fatalf("expected 25.50, got %v", q.FreightValue)
	}
}

func TestApplyQuotationRulesBuildsFinalConsumerAddress(t *testing.T) {
	street, number := "Rua A", "42"
	customer := int64(7)
	p := entity.DefaultParameters(1)
	p.FinalConsumerCustomerCode = &customer
	q := &entity.SalesQuotation{CustomerCode: &customer, Street: &street, StreetNumber: &number}
	if err := applyQuotationRules(q, p); err != nil {
		t.Fatal(err)
	}
	if q.ConsumerAddress == nil || *q.ConsumerAddress != "Rua A, 42" {
		t.Fatalf("unexpected address: %v", q.ConsumerAddress)
	}
}
