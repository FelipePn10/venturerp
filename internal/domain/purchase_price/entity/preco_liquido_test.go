package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestPrecoLiquidoEmCadeia(t *testing.T) {
	d := decimal.RequireFromString
	aj := func(seq int32, kind, calc, v string) *PriceAdjustment {
		return &PriceAdjustment{Sequence: seq, Kind: kind, CalculationType: calc, Value: d(v)}
	}
	casos := []struct {
		nome    string
		preco   string
		ajustes []*PriceAdjustment
		quer    string
	}{
		{"sem ajuste", "12.5", nil, "12.5"},
		{"10% e depois 5% em cadeia", "100", []*PriceAdjustment{aj(1, "DISCOUNT", "PERCENT", "10"), aj(2, "DISCOUNT", "PERCENT", "5")}, "85.5"},
		{"a sequência manda, não a ordem da lista", "100", []*PriceAdjustment{aj(2, "SURCHARGE", "FIXED", "2"), aj(1, "DISCOUNT", "PERCENT", "10")}, "92"},
		{"acréscimo percentual", "50", []*PriceAdjustment{aj(1, "SURCHARGE", "PERCENT", "3")}, "51.5"},
		{"desconto fixo maior que o preço zera", "5", []*PriceAdjustment{aj(1, "DISCOUNT", "FIXED", "8")}, "0"},
		{"arredonda em 6 casas", "10", []*PriceAdjustment{aj(1, "DISCOUNT", "PERCENT", "33.3333333")}, "6.666667"},
	}
	for _, c := range casos {
		if got := PrecoLiquido(d(c.preco), c.ajustes); !got.Equal(d(c.quer)) {
			t.Errorf("%s: %s, esperado %s", c.nome, got, c.quer)
		}
	}
}
