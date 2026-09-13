package machine_uc

import "testing"

// Com `bool` puro no DTO, omitir o campo é indistinguível de mandar false: quem
// gravasse um corpo parcial desligava a flag em silêncio. O contrato agora é
// ponteiro — nulo mantém, false desliga.
func TestFlagOmitidaMantemValorAtual(t *testing.T) {
	verdadeiro, falso := true, false
	casos := []struct {
		nome     string
		enviado  *bool
		atual    bool
		esperado bool
	}{
		{"omitida mantém ligada", nil, true, true},
		{"omitida mantém desligada", nil, false, false},
		{"false explícito desliga", &falso, true, false},
		{"true explícito liga", &verdadeiro, false, true},
	}
	for _, caso := range casos {
		if got := boolOuAtual(caso.enviado, caso.atual); got != caso.esperado {
			t.Errorf("%s: esperava %v, veio %v", caso.nome, caso.esperado, got)
		}
	}
}
