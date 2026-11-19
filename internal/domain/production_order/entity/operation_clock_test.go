package entity

import (
	"testing"
	"time"
)

func TestCronometroDaEtapa(t *testing.T) {
	t0 := time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
	eventos := []EventoDeEtapa{
		{Status: "IN_PROGRESS", Quando: t0},
		{Status: "PAUSED", Quando: t0.Add(2 * time.Hour)},
		{Status: "IN_PROGRESS", Quando: t0.Add(3 * time.Hour)},
	}
	inicio, ok := InicioDoTrecho(eventos)
	if !ok || !inicio.Equal(t0.Add(3*time.Hour)) {
		t.Fatalf("o trecho em andamento começa na retomada; veio %v %v", inicio, ok)
	}
	// Retomou às 11h, conclui às 12h30: 1,5 h deste trecho (as 2 h antes da pausa já entraram na pausa).
	if h := HorasDoTrecho(inicio, t0.Add(4*time.Hour+30*time.Minute)); h != 1.5 {
		t.Fatalf("esperava 1,5 h; veio %v", h)
	}
	if h := HorasDoTrecho(t0, t0.Add(20*time.Minute)); h != 0.3333 {
		t.Fatalf("20 min = 0,3333 h; veio %v", h)
	}
	if h := HorasDoTrecho(t0, t0.Add(-time.Minute)); h != 0 {
		t.Fatalf("relógio que volta dá zero; veio %v", h)
	}
	if _, ok := InicioDoTrecho([]EventoDeEtapa{{Status: "PENDING", Quando: t0}}); ok {
		t.Fatal("sem evento de andamento não há trecho")
	}
	if HorasPorExtenso(15.04) != "15,0 h" {
		t.Fatalf("horas por extenso: %q", HorasPorExtenso(15.04))
	}
	if TrechoEsquecido(12) || !TrechoEsquecido(12.01) {
		t.Fatal("o limite do cronômetro é 12 h")
	}
	for _, c := range []struct {
		de, para string
		fecha    bool
	}{{"IN_PROGRESS", "DONE", true}, {"IN_PROGRESS", "PAUSED", true}, {"IN_PROGRESS", "INTERRUPTED", true}, {"PENDING", "IN_PROGRESS", false}, {"PAUSED", "IN_PROGRESS", false}} {
		if FechaTrecho(c.de, c.para) != c.fecha {
			t.Fatalf("FechaTrecho(%s, %s) = %v", c.de, c.para, !c.fecha)
		}
	}
}
