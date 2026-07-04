package entity

import "testing"

func TestCriticalPath_LinearFallbackNoEdges(t *testing.T) {
	// 3 operations, NO network edges → must chain linearly by sequence.
	ops := []*RouteOperation{
		{ID: 30, Sequence: 30, EffTime: OperationTime{Run: 5}},
		{ID: 10, Sequence: 10, EffTime: OperationTime{Run: 2}},
		{ID: 20, Sequence: 20, EffTime: OperationTime{Run: 3}},
	}
	res := CriticalPath(ops, nil, 1)
	if !approx(res.TotalHours, 10) { // 2 + 3 + 5
		t.Fatalf("linear fallback total = %v, want 10", res.TotalHours)
	}
	want := []int64{10, 20, 30}
	if len(res.CriticalPath) != 3 {
		t.Fatalf("path = %v, want %v", res.CriticalPath, want)
	}
	for i, id := range want {
		if res.CriticalPath[i] != id {
			t.Errorf("path[%d] = %d, want %d", i, res.CriticalPath[i], id)
		}
	}
}

func TestCriticalPath_RequiresOperatorCancelsOverlap(t *testing.T) {
	// 1 → 2 with 50% overlap, but op1's work center requires an operator.
	ops := []*RouteOperation{
		{ID: 1, Sequence: 10, RequiresOperator: true, EffTime: OperationTime{Run: 10}},
		{ID: 2, Sequence: 20, EffTime: OperationTime{Run: 10}},
	}
	edges := []*NetworkEdge{{PredecessorID: 1, SuccessorID: 2, OverlapPct: 50}}
	// Overlap forced to 0 → 10 + 10 = 20 (not 15).
	if res := CriticalPath(ops, edges, 1); !approx(res.TotalHours, 20) {
		t.Errorf("requires-operator total = %v, want 20", res.TotalHours)
	}

	// Without requires-operator the overlap applies: 10*0.5 + 10 = 15.
	ops[0].RequiresOperator = false
	if res := CriticalPath(ops, edges, 1); !approx(res.TotalHours, 15) {
		t.Errorf("with overlap total = %v, want 15", res.TotalHours)
	}
}

func TestCriticalPath_QuantityScalesRun(t *testing.T) {
	// Linear 10 → 20, run 0.5h/pc, setup 1h each. qty 100.
	ops := []*RouteOperation{
		{ID: 1, Sequence: 10, EffTime: OperationTime{Setup: 1, Run: 0.5, RunBaseQty: 1}},
		{ID: 2, Sequence: 20, EffTime: OperationTime{Setup: 1, Run: 0.5, RunBaseQty: 1}},
	}
	// each op at qty 100 = 1 + 0.5*100 = 51; chained = 102.
	if res := CriticalPath(ops, nil, 100); !approx(res.TotalHours, 102) {
		t.Errorf("qty-scaled total = %v, want 102", res.TotalHours)
	}
}

func TestCriticalPath_Empty(t *testing.T) {
	if res := CriticalPath(nil, nil, 1); res.TotalHours != 0 || len(res.CriticalPath) != 0 {
		t.Errorf("empty route = %+v, want zero", res)
	}
}
