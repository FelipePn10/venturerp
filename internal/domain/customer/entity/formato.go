package entity

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Reais formata como o usuário lê: R$ 1.709.695,37.
func Reais(v decimal.Decimal) string { return "R$ " + Numero(v, 2) }

// Numero formata em pt-BR com separador de milhar: 1.234,5.
func Numero(v decimal.Decimal, casas int32) string {
	txt := v.Abs().StringFixed(casas)
	inteiro, frac, _ := strings.Cut(txt, ".")
	var b strings.Builder
	if v.IsNegative() && !v.Round(casas).IsZero() {
		b.WriteByte('-')
	}
	for i, c := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if frac != "" {
		b.WriteString("," + frac)
	}
	return b.String()
}
