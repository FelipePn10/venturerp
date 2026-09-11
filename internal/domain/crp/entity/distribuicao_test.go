package entity

import (
	"testing"
	"time"
)

func seg(dia int) time.Time { return time.Date(2026, 9, dia, 0, 0, 0, 0, time.UTC) }

// TestDistribuiCargaEspalhaPelosDias trava o comportamento que faltava: a carga
// de uma operação longa era jogada inteira no dia da ordem, criando um pico
// falso e escondendo a ocupação dos dias seguintes.
func TestDistribuiCargaEspalhaPelosDias(t *testing.T) {
	// 20 h numa capacidade de 8 h/dia = 8 + 8 + 4 em três dias úteis.
	got := DistribuiCarga(seg(7), 20, 8) // 7/9/2026 é segunda-feira
	if len(got) != 3 {
		t.Fatalf("dias = %d, esperado 3", len(got))
	}
	esperado := []float64{8, 8, 4}
	total := 0.0
	for i, c := range got {
		if c.Horas != esperado[i] {
			t.Fatalf("dia %d = %.1f h, esperado %.1f", i+1, c.Horas, esperado[i])
		}
		total += c.Horas
	}
	if total != 20 {
		t.Fatalf("total distribuído = %.1f h, esperado 20", total)
	}
}

// TestDistribuiCargaPulaFimDeSemana: capacidade de sábado e domingo é zero.
func TestDistribuiCargaPulaFimDeSemana(t *testing.T) {
	// 11/9/2026 é sexta. 16 h a 8 h/dia devem cair na sexta e na segunda.
	got := DistribuiCarga(seg(11), 16, 8)
	if len(got) != 2 {
		t.Fatalf("dias = %d, esperado 2", len(got))
	}
	if got[0].Data.Weekday() != time.Friday {
		t.Fatalf("primeiro dia = %v, esperado sexta", got[0].Data.Weekday())
	}
	if got[1].Data.Weekday() != time.Monday {
		t.Fatalf("segundo dia = %v, esperado segunda (fim de semana pulado)", got[1].Data.Weekday())
	}
}

// TestDistribuiCargaSemCapacidade: sem capacidade cadastrada não há base para
// espalhar; a carga fica num dia só, e o CRP reporta o centro à parte.
func TestDistribuiCargaSemCapacidade(t *testing.T) {
	got := DistribuiCarga(seg(7), 20, 0)
	if len(got) != 1 || got[0].Horas != 20 {
		t.Fatalf("esperado um único dia com 20 h, veio %+v", got)
	}
}

// TestDistribuiCargaComecandoNoSabado empurra para a segunda.
func TestDistribuiCargaComecandoNoSabado(t *testing.T) {
	got := DistribuiCarga(seg(12), 4, 8) // 12/9/2026 é sábado
	if len(got) != 1 || got[0].Data.Weekday() != time.Monday {
		t.Fatalf("esperado segunda-feira, veio %+v", got)
	}
}
