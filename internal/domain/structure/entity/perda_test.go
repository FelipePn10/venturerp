package entity

import (
	"math"
	"testing"
)

// O caso real que expôs a divergência: 50 chassis × 45 chapas = 2250, perda 5%.
func TestQuantidadeComPerdaTemUmaContaSo(t *testing.T) {
	const base, perda = 2250.0, 5.0
	casos := map[int]float64{
		FormulaPerdaMultiplica: 2362.5,
		FormulaPerdaDivide:     2368.421052631579,
		FormulaPerdaIgnora:     2250,
	}
	for formula, esperado := range casos {
		if got := QuantidadeComPerda(base, perda, formula); math.Abs(got-esperado) > 1e-9 {
			t.Errorf("fórmula %d: esperava %v, veio %v", formula, esperado, got)
		}
	}
	// O padrão precisa ser o mesmo que o MRP usa, senão o planejamento compra
	// uma quantidade e a ordem consome outra.
	if QuantidadeComPerda(base, perda, FormulaPerdaPadrao) != QuantidadeComPerda(base, perda, FormulaPerdaDivide) {
		t.Fatal("o padrão divergiu da fórmula 2")
	}
	// Fórmula desconhecida cai no padrão, nunca em "ignora perda".
	if QuantidadeComPerda(base, perda, 99) != QuantidadeComPerda(base, perda, FormulaPerdaPadrao) {
		t.Fatal("fórmula desconhecida não caiu no padrão")
	}
	if QuantidadeComPerda(base, 0, FormulaPerdaDivide) != base {
		t.Fatal("sem perda a quantidade não pode mudar")
	}
}
