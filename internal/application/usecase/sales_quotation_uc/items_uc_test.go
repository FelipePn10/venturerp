package sales_quotation_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type salesQuotationAllowAuth struct{ ports.AuthService }

func (salesQuotationAllowAuth) CanUpdateSalesOrder(context.Context) bool { return true }

func TestCalcItemTotals(t *testing.T) {
	item := &entity.SalesQuotationItem{
		RequestedQty: 10,
		UnitPrice:    25,
		DiscountPct:  10,
		IPIPct:       5,
		STPct:        2,
		AttendedQty:  3,
		CancelledQty: 1,
	}

	calcItemTotals(item)

	if item.TotalGross != 250 {
		t.Fatalf("expected gross 250, got %v", item.TotalGross)
	}
	if item.TotalNet != 225 {
		t.Fatalf("expected net 225, got %v", item.TotalNet)
	}
	if item.TotalNetWithIPI != 240.75 {
		t.Fatalf("expected net with taxes 240.75, got %v", item.TotalNetWithIPI)
	}
	if item.Balance != 6 {
		t.Fatalf("expected balance 6, got %v", item.Balance)
	}
}

func TestUpdateItemRejectsQuantityOverflow(t *testing.T) {
	uc := &UseCase{Auth: salesQuotationAllowAuth{}}

	_, err := uc.UpdateItem(context.Background(), request.UpdateSalesQuotationItemDTO{
		Code:         1,
		RequestedQty: 5,
		AttendedQty:  4,
		CancelledQty: 2,
	})

	if err == nil || !strings.Contains(err.Error(), "cannot exceed requested_qty") {
		t.Fatalf("expected quantity overflow validation error, got %v", err)
	}
}
