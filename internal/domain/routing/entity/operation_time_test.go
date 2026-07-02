package entity

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestResolveOperationTime_InheritAndOverride(t *testing.T) {
	def := TimeComponents{
		Setup: 1, Run: 2, Labor: 0, RunBaseQty: 1,
		Queue: 3, Wait: 4, Move: 5, CrewSize: 1, Unit: TimeUnitHour,
	}
	// No overrides → inherit everything (already in hours).
	got := ResolveOperationTime(TimeOverrides{}, def)
	if !approx(got.Setup, 1) || !approx(got.Run, 2) || !approx(got.Queue, 3) ||
		!approx(got.Wait, 4) || !approx(got.Move, 5) || !approx(got.RunBaseQty, 1) || !approx(got.CrewSize, 1) {
		t.Fatalf("inherit mismatch: %+v", got)
	}

	// Override run only.
	run := 9.0
	got = ResolveOperationTime(TimeOverrides{Run: &run}, def)
	if !approx(got.Run, 9) || !approx(got.Setup, 1) {
		t.Fatalf("override run mismatch: %+v", got)
	}
}

func TestResolveOperationTime_UnitConversion(t *testing.T) {
	// Defaults measured in minutes: 30 min run → 0.5 h.
	def := TimeComponents{Setup: 60, Run: 30, RunBaseQty: 1, CrewSize: 1, Unit: TimeUnitMinute}
	got := ResolveOperationTime(TimeOverrides{}, def)
	if !approx(got.Setup, 1) { // 60 min = 1 h
		t.Errorf("setup = %v h, want 1", got.Setup)
	}
	if !approx(got.Run, 0.5) { // 30 min = 0.5 h
		t.Errorf("run = %v h, want 0.5", got.Run)
	}

	// Override supplied in DIA (8 h/day): 0.25 day = 2 h.
	unit := TimeUnitDay
	q := 0.25
	got = ResolveOperationTime(TimeOverrides{Queue: &q, Unit: &unit}, def)
	if !approx(got.Queue, 2) {
		t.Errorf("queue = %v h, want 2 (0.25 day)", got.Queue)
	}
}

func TestOperationTime_BatchesAndMachineHours(t *testing.T) {
	// base qty 10, run 0.5 h/cycle, setup 1 h.
	ot := OperationTime{Setup: 1, Run: 0.5, RunBaseQty: 10, CrewSize: 1}
	if b := ot.Batches(73); !approx(b, 8) { // ceil(73/10)=8
		t.Errorf("batches(73) = %v, want 8", b)
	}
	// machine = setup + run*batches = 1 + 0.5*8 = 5.
	if m := ot.MachineHours(73); !approx(m, 5) {
		t.Errorf("machineHours(73) = %v, want 5", m)
	}
	// setup does not scale: qty 1 → 1 + 0.5*1 = 1.5.
	if m := ot.MachineHours(1); !approx(m, 1.5) {
		t.Errorf("machineHours(1) = %v, want 1.5", m)
	}
	// qty 0 → no batches, only nothing (setup still applies per lot? no lot → 0 batches).
	if m := ot.MachineHours(0); !approx(m, 1) {
		t.Errorf("machineHours(0) = %v, want 1 (setup only)", m)
	}
}

func TestOperationTime_LaborHoursCrewAndFallback(t *testing.T) {
	// Labor unset ⇒ equals run. crew 2 doubles.
	ot := OperationTime{Setup: 1, Run: 2, RunBaseQty: 1, CrewSize: 2}
	// (setup + run*ceil(3/1)) * crew = (1 + 6) * 2 = 14.
	if l := ot.LaborHours(3); !approx(l, 14) {
		t.Errorf("laborHours(3) = %v, want 14", l)
	}
	// Explicit labor smaller than run.
	ot2 := OperationTime{Setup: 0, Run: 5, Labor: 1, RunBaseQty: 1, CrewSize: 1}
	if l := ot2.LaborHours(4); !approx(l, 4) { // 0 + 1*4
		t.Errorf("laborHours(4) = %v, want 4", l)
	}
}

func TestOperationTime_LeadTimeScalesWithQty(t *testing.T) {
	ot := OperationTime{Setup: 1, Run: 0.5, RunBaseQty: 1, Queue: 2, Wait: 1, Move: 0.5}
	// qty 1 → 1 + 0.5*1 + 2 + 1 + 0.5 = 5.
	if lt := ot.LeadTimeHours(1); !approx(lt, 5) {
		t.Errorf("leadTime(1) = %v, want 5", lt)
	}
	// qty 100 → run scales, fixed parts don't: 1 + 0.5*100 + 3.5 = 54.5.
	if lt := ot.LeadTimeHours(100); !approx(lt, 54.5) {
		t.Errorf("leadTime(100) = %v, want 54.5", lt)
	}
	if ot.LeadTimeHours(100) <= ot.LeadTimeHours(1) {
		t.Error("lead time must grow with quantity")
	}
}
