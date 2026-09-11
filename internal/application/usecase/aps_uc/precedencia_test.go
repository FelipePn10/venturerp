package aps_uc

import (
	"testing"

	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

func op(id int64, seq int) apsrepo.OpRow { return apsrepo.OpRow{ID: id, Sequence: seq} }

func ids(rows []apsrepo.OpRow) []int64 {
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

// TestOrdenaPorPrecedenciaSemArestas: roteiro linear mantém a ordem de sequência.
func TestOrdenaPorPrecedenciaSemArestas(t *testing.T) {
	ops := []apsrepo.OpRow{op(3, 30), op(1, 10), op(2, 20)}
	ordenadas, ciclo := ordenaPorPrecedencia(ops, nil)
	if len(ciclo) > 0 {
		t.Fatalf("ciclo inesperado: %v", ciclo)
	}
	got := ids(ordenadas)
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("ordem = %v, esperado [1 2 3] por sequência", got)
	}
}

// TestOrdenaPorPrecedenciaRespeitaRede: com dois ramos paralelos, cada um só
// depois do predecessor — e a operação final depois de ambos.
func TestOrdenaPorPrecedenciaRespeitaRede(t *testing.T) {
	ops := []apsrepo.OpRow{op(1, 10), op(2, 20), op(3, 30), op(4, 40)}
	edges := []apsrepo.OpEdge{
		{PredecessorID: 1, SuccessorID: 2},
		{PredecessorID: 1, SuccessorID: 3},
		{PredecessorID: 2, SuccessorID: 4},
		{PredecessorID: 3, SuccessorID: 4},
	}
	ordenadas, ciclo := ordenaPorPrecedencia(ops, edges)
	if len(ciclo) > 0 {
		t.Fatalf("ciclo inesperado: %v", ciclo)
	}
	pos := map[int64]int{}
	for i, r := range ordenadas {
		pos[r.ID] = i
	}
	if pos[1] > pos[2] || pos[1] > pos[3] {
		t.Fatalf("operação 1 deveria vir antes de 2 e 3: %v", ids(ordenadas))
	}
	if pos[4] < pos[2] || pos[4] < pos[3] {
		t.Fatalf("operação 4 deveria vir depois de 2 e 3: %v", ids(ordenadas))
	}
}

// TestOrdenaPorPrecedenciaAcusaCiclo: rede em ciclo é recusada em vez de
// produzir uma programação silenciosamente incompleta.
func TestOrdenaPorPrecedenciaAcusaCiclo(t *testing.T) {
	ops := []apsrepo.OpRow{op(1, 10), op(2, 20), op(3, 30)}
	edges := []apsrepo.OpEdge{
		{PredecessorID: 1, SuccessorID: 2},
		{PredecessorID: 2, SuccessorID: 3},
		{PredecessorID: 3, SuccessorID: 2},
	}
	_, ciclo := ordenaPorPrecedencia(ops, edges)
	if len(ciclo) != 2 {
		t.Fatalf("operações em ciclo = %v, esperado as duas do laço", ciclo)
	}
}

// TestOrdenaPorPrecedenciaIgnoraArestaForaDaOrdem: aresta cuja ponta já foi
// concluída (não veio na lista) não pode travar a sucessora.
func TestOrdenaPorPrecedenciaIgnoraArestaForaDaOrdem(t *testing.T) {
	ops := []apsrepo.OpRow{op(2, 20), op(3, 30)}
	edges := []apsrepo.OpEdge{
		{PredecessorID: 1, SuccessorID: 2}, // 1 já concluída
		{PredecessorID: 2, SuccessorID: 3},
	}
	ordenadas, ciclo := ordenaPorPrecedencia(ops, edges)
	if len(ciclo) > 0 {
		t.Fatalf("ciclo inesperado: %v", ciclo)
	}
	if len(ordenadas) != 2 || ordenadas[0].ID != 2 {
		t.Fatalf("ordem = %v, esperado começar por 2", ids(ordenadas))
	}
}
