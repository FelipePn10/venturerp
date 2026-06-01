package service

import (
	"math"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

func TestCheckUnitCompatibility(t *testing.T) {
	cases := []struct {
		name       string
		item       types.TypeUnitOfMeasurementItem
		machine    types.MachineCapacityUnit
		wantFactor float64
		wantErr    bool
	}{
		{"KG→Kilogram 1:1", types.KG, types.Kilogram, 1.0, false},
		{"KG→Ton 0.001", types.KG, types.Ton, 0.001, false},
		{"TON→Kilogram 1000", types.TONELADA, types.Kilogram, 1000.0, false},
		{"M→Meters 1:1", types.M, types.Meters, 1.0, false},
		{"MM→Meters 0.001", types.MM, types.Meters, 0.001, false},
		{"IN→Meters 0.0254", types.IN, types.Meters, 0.0254, false},
		{"M3→Liters 1000", types.M3, types.Liters, 1000.0, false},
		{"UN→Units 1:1", types.UN, types.Units, 1.0, false},
		{"incompatible mass→length", types.KG, types.Meters, 0, true},
		{"incompatible length→mass", types.M, types.Kilogram, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := CheckUnitCompatibility(tc.item, tc.machine)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected incompatibility error, got factor %v", res.Factor)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(res.Factor-tc.wantFactor) > 1e-9 {
				t.Errorf("factor = %v, want %v", res.Factor, tc.wantFactor)
			}
		})
	}
}
