package sales_goal_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func TestGoalItemRequiresExactlyOneTarget(t *testing.T) {
	item := int64(10)
	classification := int64(20)
	_, err := goalItemFromDTO(request.SalesGoalItemDTO{GoalCode: 1, ItemCode: &item, ItemClassificationCode: &classification})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGoalItemInfersTargetType(t *testing.T) {
	item := int64(10)
	got, err := goalItemFromDTO(request.SalesGoalItemDTO{GoalCode: 1, ItemCode: &item, TargetValue: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TargetType != "ITEM" {
		t.Fatalf("target type = %s, want ITEM", got.TargetType)
	}
}

func TestPeriodRejectsInvalidDateRange(t *testing.T) {
	_, err := periodFromDTO(request.CreateSalesGoalPeriodDTO{Description: "Julho", StartDate: "2026-07-31", EndDate: "2026-07-01"})
	if err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestNormalizeAnalysisBase(t *testing.T) {
	if normalizeAnalysisBase("faturamento") != "INVOICING" {
		t.Fatal("expected faturamento to normalize to INVOICING")
	}
	if normalizeAnalysisBase("vendas") != "SALES" {
		t.Fatal("expected vendas to normalize to SALES")
	}
}
