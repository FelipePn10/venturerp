package entity

import "testing"

func TestCycleCountTransitions(t *testing.T) {
	valid := [][2]CycleCountState{{CycleScheduled, CycleCounting}, {CycleCounting, CycleDivergent}, {CycleDivergent, CycleCompleted}, {CycleCompleted, CycleApproved}}
	for _, v := range valid {
		if !CanTransition(v[0], v[1]) {
			t.Fatalf("transição válida rejeitada: %s -> %s", v[0], v[1])
		}
	}
	if CanTransition(CycleApproved, CycleCounting) {
		t.Fatal("contagem aprovada reaberta")
	}
}
