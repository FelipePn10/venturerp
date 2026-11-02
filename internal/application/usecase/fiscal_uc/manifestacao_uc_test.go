package fiscal_uc

import "testing"

func TestValidarInutilizacao(t *testing.T) {
	ok := InutilizarNumeracaoDTO{Serie: 1, NumeroInicial: 10, NumeroFinal: 12, Justificativa: "numeração pulada por falha"}
	if err := validarInutilizacao(ok); err != nil {
		t.Fatalf("faixa válida recusada: %v", err)
	}
	for nome, dto := range map[string]InutilizarNumeracaoDTO{
		"justificativa curta": {Serie: 1, NumeroInicial: 1, NumeroFinal: 1, Justificativa: "curta"},
		"faixa invertida":     {Serie: 1, NumeroInicial: 5, NumeroFinal: 4, Justificativa: "numeração pulada por falha"},
		"número zero":         {Serie: 1, NumeroInicial: 0, NumeroFinal: 4, Justificativa: "numeração pulada por falha"},
		"série alta":          {Serie: 1000, NumeroInicial: 1, NumeroFinal: 4, Justificativa: "numeração pulada por falha"},
	} {
		if err := validarInutilizacao(dto); err == nil {
			t.Errorf("%s: deveria ser recusada", nome)
		}
	}
}
