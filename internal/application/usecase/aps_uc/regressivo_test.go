package aps_uc

import (
	"testing"
	"time"
)

// TestRecuaPorHorasUteis: o espelho de advanceByWorkHours. Sem ele o
// sequenciamento só sabia responder "começando hoje, termino quando".
func TestRecuaPorHorasUteis(t *testing.T) {
	// Quarta 9/9/2026 às 00h; 16 h a 8 h/dia começam na segunda 7/9.
	fim := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	got := recuaPorHorasUteis(fim, 16, 8)
	esperado := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	if !got.Equal(esperado) {
		t.Fatalf("início = %s, esperado %s", got.Format("2006-01-02 15:04"), esperado.Format("2006-01-02 15:04"))
	}
}

// TestRecuaPorHorasUteisPulaFimDeSemana: recuando de segunda, o trabalho cai na
// sexta anterior, não no domingo.
func TestRecuaPorHorasUteisPulaFimDeSemana(t *testing.T) {
	fim := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC) // segunda
	got := recuaPorHorasUteis(fim, 8, 8)
	if got.Weekday() == time.Saturday || got.Weekday() == time.Sunday {
		t.Fatalf("início caiu em %v; o recuo deve pular o fim de semana", got.Weekday())
	}
	if got.After(fim) {
		t.Fatalf("início %s não pode ser depois do fim %s", got, fim)
	}
}

// TestRecuaEAvancaSaoEspelhos: avançar a partir do início recuado devolve o fim.
func TestRecuaEAvancaSaoEspelhos(t *testing.T) {
	fim := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC) // sexta
	inicio := recuaPorHorasUteis(fim, 12, 8)
	volta := advanceByWorkHours(inicio, 12, 8)
	if volta.Sub(fim) > time.Minute || fim.Sub(volta) > time.Minute {
		t.Fatalf("ida e volta divergem: recuo deu %s, avanço devolveu %s (esperado %s)",
			inicio.Format("02/01 15:04"), volta.Format("02/01 15:04"), fim.Format("02/01 15:04"))
	}
}

// TestRetiraDiaNaoUtil recua sábado e domingo para a sexta.
func TestRetiraDiaNaoUtil(t *testing.T) {
	sabado := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	if d := retiraDiaNaoUtil(sabado); d.Weekday() != time.Friday {
		t.Fatalf("sábado recuou para %v, esperado sexta", d.Weekday())
	}
	domingo := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	if d := retiraDiaNaoUtil(domingo); d.Weekday() != time.Friday {
		t.Fatalf("domingo recuou para %v, esperado sexta", d.Weekday())
	}
}
