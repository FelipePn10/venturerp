package procurement_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func ptrInt64(v int64) *int64 { return &v }

func TestPlanInspectionStockLegs(t *testing.T) {
	const inspectionWH = int64(10)

	t.Run("routes conform, rework and rejected to distinct warehouses", func(t *testing.T) {
		dto := request.AnalyzeReceivingInspectionOrderDTO{
			ConformQty:             8,
			ReworkQty:              1,
			RejectedQty:            1,
			DestinationWarehouseID: ptrInt64(1),
			ReworkWarehouseID:      ptrInt64(2),
			RejectionWarehouseID:   ptrInt64(3),
		}
		legs := planInspectionStockLegs(inspectionWH, dto)
		if len(legs) != 3 {
			t.Fatalf("expected 3 legs, got %d: %+v", len(legs), legs)
		}
		want := map[int64]float64{1: 8, 2: 1, 3: 1}
		for _, l := range legs {
			if want[l.To] != l.Qty {
				t.Errorf("warehouse %d: expected qty %v, got %v", l.To, want[l.To], l.Qty)
			}
		}
	})

	t.Run("restricted falls back to destination when no restricted warehouse", func(t *testing.T) {
		dto := request.AnalyzeReceivingInspectionOrderDTO{
			ConformQty:             5,
			RestrictedQty:          2,
			DestinationWarehouseID: ptrInt64(1),
		}
		legs := planInspectionStockLegs(inspectionWH, dto)
		if len(legs) != 2 {
			t.Fatalf("expected 2 legs, got %d: %+v", len(legs), legs)
		}
		for _, l := range legs {
			if l.To != 1 {
				t.Errorf("expected restricted+conform to route to destination 1, got %d", l.To)
			}
		}
	})

	t.Run("skips zero quantities, missing targets and self-transfers", func(t *testing.T) {
		dto := request.AnalyzeReceivingInspectionOrderDTO{
			ConformQty:             4,
			RejectedQty:            3,
			RestrictedQty:          0,
			DestinationWarehouseID: ptrInt64(inspectionWH), // same as source -> skipped
			RejectionWarehouseID:   nil,                    // missing target -> skipped
		}
		legs := planInspectionStockLegs(inspectionWH, dto)
		if len(legs) != 0 {
			t.Fatalf("expected 0 legs, got %d: %+v", len(legs), legs)
		}
	})
}

func TestRatioScore(t *testing.T) {
	cases := []struct {
		name     string
		num, den float64
		want     float64
	}{
		{"no data defaults to 100", 0, 0, 100},
		{"perfect", 100, 100, 100},
		{"half", 5, 10, 50},
		{"clamped to 100 when num>den", 12, 10, 100},
		{"clamped to 0 when negative", -3, 10, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ratioScore(c.num, c.den); got != c.want {
				t.Errorf("ratioScore(%v,%v)=%v want %v", c.num, c.den, got, c.want)
			}
		})
	}
}

func TestOverallIQF(t *testing.T) {
	// quality 40% + delivery 30% + commercial 20% + service 10%
	got := overallIQF(100, 100, 100, 100)
	if got != 100 {
		t.Fatalf("all-100 overall = %v, want 100", got)
	}
	got = overallIQF(50, 100, 100, 100) // 20 + 30 + 20 + 10
	if got != 80 {
		t.Fatalf("quality-50 overall = %v, want 80", got)
	}
}
