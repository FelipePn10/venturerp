package technical_assistance_uc

import (
	"testing"
	"time"
)

func TestMomentoDaAbertura(t *testing.T) {
	agora := time.Date(2026, 10, 25, 15, 40, 0, 0, time.Local)
	hoje := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) // como o ParseDate devolve "2026-10-25"
	if got := momentoDaAbertura(hoje, agora); !got.Equal(agora) {
		t.Fatalf("aberto hoje vale a hora real: %v", got)
	}
	ontem := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	if got := momentoDaAbertura(ontem, agora); !got.Equal(ontem) {
		t.Fatalf("data retroativa fica: %v", got)
	}
	comHora := time.Date(2026, 10, 25, 9, 30, 0, 0, time.UTC)
	if got := momentoDaAbertura(comHora, agora); !got.Equal(comHora) {
		t.Fatalf("hora informada fica: %v", got)
	}
}
