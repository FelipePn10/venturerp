package valueobject

import "testing"

func TestGTINValido(t *testing.T) {
	for gtin, esperado := range map[string]bool{
		"7891000100103":  true,  // GTIN-13
		"7891000100104":  false, // dígito verificador errado
		"96385074":       true,  // GTIN-8
		"96385075":       false,
		"036000291452":   true,  // GTIN-12
		"17891000100100": true,  // GTIN-14
		"789100010010":   false, // 12 dígitos com verificador errado
		"789100010":      false, // tamanho que não existe
		"":               false,
		"789100010010A":  false,
		"SEM GTIN":       false,
	} {
		if GTINValido(gtin) != esperado {
			t.Errorf("GTINValido(%q) deveria ser %v", gtin, esperado)
		}
	}
}
