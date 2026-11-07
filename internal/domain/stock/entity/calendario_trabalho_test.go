package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDiasDeConsumoSegueOCalendario(t *testing.T) {
	sexta := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC) // sexta-feira
	semExcecao := CalendarioDeTrabalho{}
	// 5 dias úteis de prazo a partir da sexta: seg a sex seguintes = 5 dias de consumo.
	if n := semExcecao.DiasDeConsumoNoPrazo(sexta, 5); n != 5 {
		t.Fatalf("sem exceção: %d", n)
	}
	comSabado := CalendarioDeTrabalho{Excecoes: map[string]bool{"2026-11-07": true}}
	if n := comSabado.DiasDeConsumoNoPrazo(sexta, 5); n != 6 {
		t.Fatalf("sábado trabalhado no prazo soma um dia: %d", n)
	}
	comFeriado := CalendarioDeTrabalho{Excecoes: map[string]bool{"2026-11-20": false}}
	if n := comFeriado.DiasDeConsumoNoPrazo(sexta, 10); n != 9 {
		t.Fatalf("feriado parado no prazo tira um dia: %d", n)
	}
	// 90 dias de histórico com 2 sábados trabalhados.
	cal := CalendarioDeTrabalho{Excecoes: map[string]bool{"2026-10-03": true, "2026-10-10": true}}
	if n := cal.DiasTrabalhados(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)); n != 24 {
		t.Fatalf("outubro/2026 tem 22 dias úteis + 2 sábados: %d", n)
	}
	if p := PontoPorDiaTrabalhado(decimal.RequireFromString("12.5"), 6, decimal.NewFromInt(20)); !p.Equal(decimal.NewFromInt(95)) {
		t.Fatalf("12,5 × 6 + 20 = 95: %s", p)
	}
}
