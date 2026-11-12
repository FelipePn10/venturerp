package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

// A OF e o MRP pedem o componente na escala em que ele é comprado: para cima,
// nunca abaixo do que a estrutura exige.
func TestQuantidadeDoComponenteNaEscalaDaCompra(t *testing.T) {
	d := decimal.RequireFromString
	casos := []struct{ base, porUnidade, perda, esperado string }{
		{"1", "52593.317345", "0", "52593.3174"}, // 52.593,317345 → para cima
		{"2", "3", "10", "6.6667"},               // 6 ÷ 0,9 = 6,666…
		{"10", "0.5", "0", "5"},                  // exato não muda
		{"1", "0.00001", "0", "0.0001"},          // nunca zera
		{"3", "0.1000000001", "0", "0.3"},        // resíduo além da 6ª casa não sobe
	}
	for _, c := range casos {
		got := QuantidadeDoComponente(d(c.base), d(c.porUnidade), d(c.perda), FormulaPerdaPadrao)
		if !got.Equal(d(c.esperado)) {
			t.Errorf("%s × %s (perda %s%%) = %s, esperado %s", c.base, c.porUnidade, c.perda, got, c.esperado)
		}
	}
}
