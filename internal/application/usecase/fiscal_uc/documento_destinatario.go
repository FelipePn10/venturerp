package fiscal_uc

import (
	"strings"
	"unicode"
)

// documentoSoDigitos tira a máscara do CNPJ/CPF do destinatário. O cliente
// cadastrado como "22.194.160/0001-51" fazia o faturamento falhar: o campo da nota
// tem 14 dígitos e o XML da NF-e também não aceita máscara.
func documentoSoDigitos(v *string) *string {
	if v == nil {
		return nil
	}
	d := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, *v)
	return &d
}

// inscricaoEstadual tira a máscara da IE e mantém "ISENTO".
func inscricaoEstadual(v *string) *string {
	if v == nil {
		return nil
	}
	if t := strings.ToUpper(strings.TrimSpace(*v)); strings.HasPrefix(t, "ISENT") {
		return &t
	}
	return documentoSoDigitos(v)
}
