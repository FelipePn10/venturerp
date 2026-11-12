package aps_uc

import (
	"testing"
	"time"

	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

func TestEntregaDaOrdemEANecessidadeENaoOInicio(t *testing.T) {
	inicio := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	necessidade := time.Date(2026, 12, 18, 0, 0, 0, 0, time.UTC)
	if got := (apsrepo.OrderRow{PlannedDate: inicio, DueDate: necessidade}).Entrega(); !got.Equal(necessidade) {
		t.Fatalf("entrega = %v, esperado a necessidade", got)
	}
	// Fonte que não informa a entrega: vale o início, como sempre foi.
	if got := (apsrepo.OrderRow{PlannedDate: inicio}).Entrega(); !got.Equal(inicio) {
		t.Fatalf("entrega = %v, esperado o início", got)
	}
}

func TestTerminaDepoisDaEntrega(t *testing.T) {
	entrega := time.Date(2026, 12, 18, 0, 0, 0, 0, time.UTC)
	fim := func(d int, h int) map[int64]time.Time {
		return map[int64]time.Time{1: entrega.AddDate(0, 0, -5), 2: time.Date(2026, 12, d, h, 0, 0, 0, time.UTC)}
	}
	if terminaDepoisDaEntrega(fim(18, 17), entrega) {
		t.Fatal("terminar no fim da tarde do dia da entrega está no prazo")
	}
	if !terminaDepoisDaEntrega(fim(19, 9), entrega) {
		t.Fatal("terminar no dia seguinte é atraso")
	}
	if terminaDepoisDaEntrega(fim(30, 9), time.Time{}) {
		t.Fatal("sem data de entrega não há como dizer que atrasou")
	}
	if terminaDepoisDaEntrega(nil, entrega) {
		t.Fatal("ordem sem operação programada não é atrasada")
	}
}
