package entity

import "testing"

// Etapa fantasma fica no grafo (a dependência passa por ela) mas não consome
// prazo nem dias de terceiro.
func TestEtapaFantasmaNaoConsomePrazo(t *testing.T) {
	ops := []*RouteOperation{
		{ID: 1, Sequence: 10, Situation: RouteOpApproved, EffTime: OperationTime{Run: 4}},
		{ID: 2, Sequence: 20, Situation: RouteOpGhost, EffTime: OperationTime{Run: 100}, EffectiveSubcontractDays: 30},
		{ID: 3, Sequence: 30, Situation: RouteOpApproved, EffTime: OperationTime{Run: 6}},
	}
	r := CriticalPath(ops, nil, 1)
	if r.TotalHours != 10 || r.SubcontractDays != 0 {
		t.Fatalf("10 + fantasma + 30 = 4 h + 6 h, sem terceiro; veio %v h e %d dias", r.TotalHours, r.SubcontractDays)
	}
	if r.EarlyStartHours[3] != 4 {
		t.Fatalf("a etapa 30 começa às 4 h; veio %v", r.EarlyStartHours[3])
	}
}
