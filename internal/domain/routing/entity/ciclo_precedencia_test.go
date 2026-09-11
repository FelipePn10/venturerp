package entity

import "testing"

// TestCriticalPathAcusaCiclo trava a falha em que uma rede de precedências com
// ciclo (A→B→A) devolvia 0 h e caminho vazio, em silêncio — e o MRP planejava
// como se o item não levasse tempo nenhum para ser produzido.
func TestCriticalPathAcusaCiclo(t *testing.T) {
	op := func(id int64, seq int16, setup float64) *RouteOperation {
		return &RouteOperation{ID: id, Sequence: seq, EffTime: OperationTime{Setup: setup, RunBaseQty: 1, CrewSize: 1}}
	}
	ops := []*RouteOperation{op(1, 10, 2), op(2, 20, 3), op(3, 30, 5)}
	edges := []*NetworkEdge{
		{PredecessorID: 1, SuccessorID: 2},
		{PredecessorID: 2, SuccessorID: 3},
		{PredecessorID: 3, SuccessorID: 2}, // ciclo
	}

	r := CriticalPath(ops, edges, 1)
	if !r.HasCycle() {
		t.Fatal("ciclo de precedência não foi detectado")
	}
	if len(r.CycleOperations) != 2 {
		t.Fatalf("operações em ciclo = %v, esperado as duas do laço", r.CycleOperations)
	}
	if r.TotalHours == 0 {
		t.Fatal("com ciclo o total não deve ser zero: o zero passava por lead time válido")
	}
}

// TestCriticalPathSemCicloNaoAcusa garante que a detecção não gera falso alarme
// numa rede normal, inclusive com convergência (dois caminhos para a mesma op).
func TestCriticalPathSemCicloNaoAcusa(t *testing.T) {
	op := func(id int64, seq int16, setup float64) *RouteOperation {
		return &RouteOperation{ID: id, Sequence: seq, EffTime: OperationTime{Setup: setup, RunBaseQty: 1, CrewSize: 1}}
	}
	ops := []*RouteOperation{op(1, 10, 2), op(2, 20, 3), op(3, 30, 5), op(4, 40, 1)}
	edges := []*NetworkEdge{
		{PredecessorID: 1, SuccessorID: 2},
		{PredecessorID: 1, SuccessorID: 3},
		{PredecessorID: 2, SuccessorID: 4},
		{PredecessorID: 3, SuccessorID: 4},
	}

	r := CriticalPath(ops, edges, 1)
	if r.HasCycle() {
		t.Fatalf("falso alarme de ciclo: %v", r.CycleOperations)
	}
	// Caminho crítico: 1 (2h) → 3 (5h) → 4 (1h) = 8h.
	if r.TotalHours != 8 {
		t.Fatalf("lead time = %.1f h, esperado 8 h pelo caminho mais longo", r.TotalHours)
	}
}
