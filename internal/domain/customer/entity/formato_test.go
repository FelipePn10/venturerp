package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFormatoPtBR(t *testing.T) {
	casos := []struct {
		v     string
		casas int32
		want  string
	}{
		{"1709695.37", 2, "1.709.695,37"}, {"100", 2, "100,00"}, {"1000", 0, "1.000"},
		{"13.44", 1, "13,4"}, {"-2500.5", 2, "-2.500,50"}, {"-0.001", 2, "0,00"}, {"0", 2, "0,00"},
	}
	for _, c := range casos {
		if got := Numero(decimal.RequireFromString(c.v), c.casas); got != c.want {
			t.Errorf("Numero(%s, %d) = %q, quer %q", c.v, c.casas, got, c.want)
		}
	}
	if got := Reais(decimal.RequireFromString("19000")); got != "R$ 19.000,00" {
		t.Errorf("Reais = %q", got)
	}
}
