package handler

import "testing"
import "net/url"

// TestValorDaQueryDetectaOFormato: a versão anterior apagava TODO ponto, e
// `1234.56` — o que um input numérico manda — virava `123456`.
func TestValorDaQueryDetectaOFormato(t *testing.T) {
	casos := map[string]*float64{
		"1234.56":      ptrF64(1234.56),
		"1234,56":      ptrF64(1234.56),
		"1.234,56":     ptrF64(1234.56),
		"1,234.56":     ptrF64(1234.56),
		"1234":         ptrF64(1234),
		"0.5":          ptrF64(0.5),
		"1.234.567,89": ptrF64(1234567.89),
		"":             nil,
		"abc":          nil,
	}
	for entrada, esperado := range casos {
		q := url.Values{"v": []string{entrada}}
		got := valorDaQuery(q, "v")
		if esperado == nil {
			if got != nil {
				t.Fatalf("%q devolveu %v, esperado nulo", entrada, *got)
			}
			continue
		}
		if got == nil {
			t.Fatalf("%q devolveu nulo, esperado %v", entrada, *esperado)
		}
		if *got != *esperado {
			t.Fatalf("%q lido como %v, esperado %v", entrada, *got, *esperado)
		}
	}
}

func ptrF64(v float64) *float64 { return &v }
