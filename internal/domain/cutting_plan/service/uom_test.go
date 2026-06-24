package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

func TestStockQtyForLength(t *testing.T) {
	cases := []struct {
		name    string
		uom     types.TypeUnitOfMeasurementItem
		length  float64
		factor  float64
		want    float64
		wantErr bool
	}{
		{"piece defaults to one", types.UN, 6000, 0, 1, false},
		{"empty uom defaults to one", "", 6000, 0, 1, false},
		{"meters", types.M, 6000, 0, 6, false},
		{"centimeters", types.CM, 6000, 0, 600, false},
		{"millimeters", types.MM, 6000, 0, 6000, false},
		{"inches", types.IN, 2540, 0, 100, false},
		{"kg with linear density", types.KG, 6000, 3.5, 21, false},  // 6m × 3.5 kg/m
		{"m2 with strip width", types.M2, 6000, 1.2, 7.2, false},    // 6m × 1.2 m²/m
		{"m3 with cross-section", types.M3, 2000, 0.05, 0.1, false}, // 2m × 0.05 m³/m
		{"mass without factor errors", types.KG, 6000, 0, 0, true},
		{"area without factor errors", types.M2, 6000, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := StockQtyForLength(c.uom, c.length, c.factor)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s", c.uom)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got < c.want-1e-6 || got > c.want+1e-6 {
				t.Fatalf("StockQtyForLength(%s, %v, %v) = %v, want %v", c.uom, c.length, c.factor, got, c.want)
			}
		})
	}
}
