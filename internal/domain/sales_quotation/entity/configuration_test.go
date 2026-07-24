package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestCommissionPatternRequiresExactDistribution(t *testing.T) {
	p := &CommissionPattern{Code: 1, Description: "Padrão", CommissionPct: decimal.NewFromInt(10), InvoicePct: decimal.NewFromInt(4), PaymentPct: decimal.NewFromInt(5)}
	if err := p.Validate(); err == nil {
		t.Fatal("expected invalid commission distribution")
	}
	p.PaymentPct = decimal.NewFromInt(6)
	if err := p.Validate(); err != nil {
		t.Fatalf("expected valid distribution: %v", err)
	}
}

func TestAttachmentLimit(t *testing.T) {
	a := &Attachment{SalesQuotationCode: 1, FileName: "documento.pdf", StorageKey: "tenant/quote/documento.pdf", FileSize: MaxAttachmentSize, Content: make([]byte, MaxAttachmentSize)}
	if err := a.Validate(); err != nil {
		t.Fatalf("limit must be accepted: %v", err)
	}
	a.FileSize++
	if err := a.Validate(); err == nil {
		t.Fatal("attachment over 10 MB must be rejected")
	}
}

func TestDAVOnlyAllowsOfficialReport(t *testing.T) {
	now := time.Now()
	q := &SalesQuotation{DAVGeneratedAt: &now}
	if !q.DocumentActionAllowed("DAV_REPORT") {
		t.Fatal("DAV report must remain allowed")
	}
	for _, action := range []string{"FISCAL_RECEIPT", "SALES_ORDER", "EMAIL"} {
		if q.DocumentActionAllowed(action) {
			t.Fatalf("%s must be blocked after DAV", action)
		}
	}
}
