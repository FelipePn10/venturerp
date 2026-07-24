package sales_quotation_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/shopspring/decimal"
)

type salesQuotationAllowAuth struct{ ports.AuthService }

func (salesQuotationAllowAuth) CanUpdateSalesOrder(context.Context) bool { return true }

func TestCalcItemTotals(t *testing.T) {
	item := &entity.SalesQuotationItem{
		RequestedQty: decimal.NewFromInt(10),
		UnitPrice:    decimal.NewFromInt(25),
		DiscountPct:  decimal.NewFromInt(10),
		IPIPct:       decimal.NewFromInt(5),
		STPct:        decimal.NewFromInt(2),
		AttendedQty:  decimal.NewFromInt(3),
		CancelledQty: decimal.NewFromInt(1),
	}

	calcItemTotals(item)

	if !item.TotalGross.Equal(decimal.NewFromInt(250)) {
		t.Fatalf("expected gross 250, got %v", item.TotalGross)
	}
	if !item.TotalNet.Equal(decimal.NewFromInt(225)) {
		t.Fatalf("expected net 225, got %v", item.TotalNet)
	}
	if !item.TotalNetWithIPI.Equal(decimal.RequireFromString("240.75")) {
		t.Fatalf("expected net with taxes 240.75, got %v", item.TotalNetWithIPI)
	}
	if !item.Balance.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("expected balance 6, got %v", item.Balance)
	}
}

func TestUpdateItemRejectsQuantityOverflow(t *testing.T) {
	uc := &UseCase{Auth: salesQuotationAllowAuth{}}

	_, err := uc.UpdateItem(context.Background(), request.UpdateSalesQuotationItemDTO{
		Code:         1,
		RequestedQty: decimal.NewFromInt(5),
		AttendedQty:  decimal.NewFromInt(4),
		CancelledQty: decimal.NewFromInt(2),
	})

	if err == nil || !strings.Contains(err.Error(), "cannot exceed requested_qty") {
		t.Fatalf("expected quantity overflow validation error, got %v", err)
	}
}
