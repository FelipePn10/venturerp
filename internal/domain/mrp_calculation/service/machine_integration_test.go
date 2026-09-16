package service

import (
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"math"
	"testing"
)

func TestMachineMinutesUsesItemRateOnce(t *testing.T) {
	eff := 0.8
	mt := &entity.MachineTimeInfo{ProductionTime: 1, ProductionTimeUnit: "HORA", ProductionBaseQty: 120, SetupTime: 15, EfficiencyRate: &eff, MachineEfficiencyRate: 0.5, WorkingHoursPerDay: 8, TimeBasis: "PROPORTIONAL"}
	got, err := machineMinutes(mt, 60)
	if err != nil || math.Abs(got-52.5) > 1e-9 {
		t.Fatalf("got %v,%v; want 52.5 min", got, err)
	}

	eff = 1
	got, err = machineMinutes(mt, 60)
	if err != nil || got != 45 {
		t.Fatalf("real expected rate discounted twice: %v %v", got, err)
	}
	eff = 0.8
	mt.TimeBasis = "CYCLE"
	got, err = machineMinutes(mt, 60)
	if err != nil || got != 90 {
		t.Fatalf("cycle got %v,%v; want 90", got, err)
	}
	mt.ProductionTime = 60
	mt.ProductionTimeUnit = "MINUTO"
	got, err = machineMinutes(mt, 60)
	if err != nil || got != 90 {
		t.Fatalf("minutes got %v,%v", got, err)
	}
	mt.ProductionBaseQty = 0
	if _, err = machineMinutes(mt, 60); err == nil {
		t.Fatal("accepted zero base quantity")
	}
}
func TestMachineSelectionPrefersExactMaskAndStablePriority(t *testing.T) {
	base := &entity.MachineTimeInfo{MachineID: 1, Priority: 0}
	other := &entity.MachineTimeInfo{MachineID: 2, Mask: "OTHER", Priority: 0}
	exact := &entity.MachineTimeInfo{MachineID: 3, Mask: "A", Priority: 9}
	if got := selectMachineTime([]*entity.MachineTimeInfo{base, other, exact}, "A"); got != exact {
		t.Fatal("exact mask must win")
	}
	if got := selectMachineTime([]*entity.MachineTimeInfo{base, other, exact}, "B"); got != base {
		t.Fatal("default mask expected")
	}
	if got := selectMachineTime([]*entity.MachineTimeInfo{other, exact}, "B"); got != nil {
		t.Fatal("different mask selected")
	}
}

func TestMachineMaterialDependencies(t *testing.T) {
	parent := int64(10)
	in := []*entity.PlannedOrderSuggestion{{Code: 1, ItemCode: 10}, {Code: 2, ItemCode: 20, ParentItemCode: &parent}}
	out, err := orderMachineSuggestions(in)
	if err != nil || out[0].Code != 2 {
		t.Fatalf("component must precede parent: %v %v", out, err)
	}
	child := int64(20)
	in[0].ParentItemCode = &child
	if _, err = orderMachineSuggestions(in); err == nil {
		t.Fatal("material cycle accepted")
	}
}
