package service

import (
	"math"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

func TestCalculateProductionTime_BatchAndSetup(t *testing.T) {
	imt := &entity.ItemMachineTime{
		ProductionTime:     5, // 5 minutes per cycle
		ProductionTimeUnit: types.Minute,
		ProductionBaseQty:  10, // 10 items per cycle
		SetupTime:          30,
	}
	machine := &entity.Machine{
		Capacity:       100,
		CapacityPeriod: types.Minute,
		EfficiencyRate: 1.0,
	}

	// demand 73 → ceil(73/10) = 8 cycles → 8*5 = 40 machining + 30 setup = 70 min
	res := CalculateProductionTime(imt, machine, 73, 1.0, 480)
	if res.BatchCount != 8 {
		t.Errorf("BatchCount = %v, want 8", res.BatchCount)
	}
	if res.MachiningMinutes != 40 {
		t.Errorf("MachiningMinutes = %v, want 40", res.MachiningMinutes)
	}
	if res.SetupMinutes != 30 {
		t.Errorf("SetupMinutes = %v, want 30", res.SetupMinutes)
	}
	if res.TotalMinutes != 70 {
		t.Errorf("TotalMinutes = %v, want 70", res.TotalMinutes)
	}
	if math.Abs(res.TotalHours-70.0/60.0) > 1e-9 {
		t.Errorf("TotalHours = %v", res.TotalHours)
	}
}

func TestCalculateProductionTime_HourUnitScaling(t *testing.T) {
	// production time expressed in hours should scale by 60.
	imt := &entity.ItemMachineTime{
		ProductionTime:     1, // 1 hour per cycle
		ProductionTimeUnit: types.Hour,
		ProductionBaseQty:  1,
		SetupTime:          0,
	}
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Hour, EfficiencyRate: 1}
	res := CalculateProductionTime(imt, machine, 3, 1.0, 480)
	// 3 cycles × 60 min = 180
	if res.TotalMinutes != 180 {
		t.Errorf("TotalMinutes = %v, want 180", res.TotalMinutes)
	}
}

func TestCalculateProductionTime_Bottleneck(t *testing.T) {
	imt := &entity.ItemMachineTime{ProductionTime: 1, ProductionTimeUnit: types.Minute, ProductionBaseQty: 1, SetupTime: 0}
	// machine capacity 1 unit/min, efficiency 1 → 1 unit/min.
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Minute, EfficiencyRate: 1}

	// demand 100 units, conversion 1 → required throughput 100/100min = 1/min, not a bottleneck.
	res := CalculateProductionTime(imt, machine, 100, 1.0, 480)
	if res.MachineIsBottleneck {
		t.Errorf("expected not bottleneck, got bottleneck (capPerMin=%v)", res.MachineCapacityPerMinute)
	}

	// conversion 5 → demand in machine units 500 over 100 min → 5/min > 1/min capacity → bottleneck.
	res = CalculateProductionTime(imt, machine, 100, 5.0, 480)
	if !res.MachineIsBottleneck {
		t.Errorf("expected bottleneck with high conversion factor")
	}
}

func TestCalculateProductionTime_DefaultWorkingMinutes(t *testing.T) {
	imt := &entity.ItemMachineTime{ProductionTime: 1, ProductionTimeUnit: types.Day, ProductionBaseQty: 1, SetupTime: 0}
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Day, EfficiencyRate: 1}
	// workingMins <= 0 → defaults to 480; 1 day cycle = 480 min.
	res := CalculateProductionTime(imt, machine, 1, 1.0, 0)
	if res.TotalMinutes != 480 {
		t.Errorf("TotalMinutes = %v, want 480 (default day)", res.TotalMinutes)
	}
}
