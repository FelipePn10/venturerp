package sales_quotation_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

func TestValidManualTransition(t *testing.T) {
	tests := []struct {
		name string
		from entity.SalesQuotationStatus
		to   entity.SalesQuotationStatus
		want bool
	}{
		{"draft to budget analysis", entity.SalesQuotationStatusDraft, entity.SalesQuotationStatusBudgetAnalysis, true},
		{"analysis to venture budget", entity.SalesQuotationStatusAnalysis, entity.SalesQuotationStatusVentureBudget, true},
		{"terminal cancellation requires dedicated operation", entity.SalesQuotationStatusDraft, entity.SalesQuotationStatusCancelled, false},
		{"attended is terminal", entity.SalesQuotationStatusAttended, entity.SalesQuotationStatusDraft, false},
		{"cancelled is terminal", entity.SalesQuotationStatusCancelled, entity.SalesQuotationStatusVentureBudget, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validManualTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("validManualTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
