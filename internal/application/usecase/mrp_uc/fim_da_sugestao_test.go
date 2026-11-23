package mrp_uc

import (
	"testing"
	"time"
)

// Bug F: OF só com etapa externa (sem agendamento de máquina) ficava sem fim.
func TestFimDaSugestaoSemAgendamentoUsaANecessidade(t *testing.T) {
	need := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	if fim := fimDaSugestao(nil, need); fim == nil || !fim.Equal(need) {
		t.Fatalf("fim = %v, quer a necessidade", fim)
	}
	agendado := need.AddDate(0, 0, -2)
	if fim := fimDaSugestao(&agendado, need); !fim.Equal(agendado) {
		t.Fatalf("fim = %v, quer o agendado", fim)
	}
}
