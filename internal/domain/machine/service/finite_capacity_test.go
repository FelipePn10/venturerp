package service

import (
	"testing"
	"time"
)

func TestFiniteCapacityKeepsClosedCyclesWhole(t *testing.T) {
	d := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	windows := []CapacityWindow{{d, d.Add(8 * time.Hour)}}
	busy := []CapacityWindow{{d.Add(90 * time.Minute), d.Add(2 * time.Hour)}}
	slots, err := AllocateMachineCycles(d, 2, 60, 15, windows, busy)
	if err != nil || len(slots) != 2 {
		t.Fatalf("%v %v", slots, err)
	}
	if slots[0].End != d.Add(75*time.Minute) || slots[1].Start != d.Add(2*time.Hour) || slots[1].End != d.Add(3*time.Hour) {
		t.Fatalf("split cycle: %v", slots)
	}
	if _, err = AllocateMachineCycles(d, 1, 600, 0, windows, nil); err == nil {
		t.Fatal("cycle longer than shift accepted")
	}
	continuous, err := AllocateMachineTime(d, 135, windows, busy)
	if err != nil || continuous[1].End != d.Add(165*time.Minute) {
		t.Fatalf("continuous rate must use remaining gap: %v %v", continuous, err)
	}
}

// Terceiro turno 22:00–06:00. Enquanto o banco proibia fim menor que início, a
// única forma de cadastrar era quebrar em 22:00–23:59 e 00:00–06:00 — e aí um
// ciclo fechado de três horas não cabia em nenhuma das duas metades, embora a
// máquina estivesse produzindo a madrugada inteira. A janela contínua resolve.
func TestFiniteCapacityAceitaTurnoQueViraODia(t *testing.T) {
	inicio := time.Date(2026, 9, 14, 22, 0, 0, 0, time.UTC)
	noturno := []CapacityWindow{{inicio, inicio.Add(8 * time.Hour)}} // até 06:00 do dia seguinte

	slots, err := AllocateMachineCycles(inicio, 2, 180, 0, noturno, nil)
	if err != nil {
		t.Fatalf("ciclo de 3h deveria caber na madrugada: %v", err)
	}
	fim := slots[len(slots)-1].End
	if esperado := inicio.Add(6 * time.Hour); !fim.Equal(esperado) {
		t.Fatalf("término %v, esperado %v", fim, esperado)
	}
	if fim.Day() == inicio.Day() {
		t.Fatalf("o turno deveria atravessar a meia-noite, terminou em %v", fim)
	}

	// Contraprova: partido em duas janelas (o contorno antigo), o mesmo ciclo
	// não cabe — é exatamente a capacidade que se perdia.
	partido := []CapacityWindow{
		{inicio, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC).Add(1 * time.Second), time.Date(2026, 9, 15, 6, 0, 0, 0, time.UTC)},
	}
	if _, err := AllocateMachineCycles(inicio, 2, 180, 0, partido, nil); err == nil {
		t.Fatal("com a janela partida o ciclo de 3h não deveria caber duas vezes")
	}
}
