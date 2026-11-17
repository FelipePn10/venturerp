package validation

// SuframaValida confere a inscrição SUFRAMA: 9 dígitos, o último pelo módulo
// 11 (pesos 9 a 2; resto 0 ou 1 → 0). É a conferência da SEFAZ (rejeição 235):
// em homologação 200100122 foi aceita e 200100123 recusada.
func SuframaValida(inscricao string) bool {
	d := stripNonDigits(inscricao)
	if len(d) != 9 {
		return false
	}
	return mod11(soma(d, decrescente(9)...)) == dig(d, 8)
}
