package entity

import (
	"strings"
	"testing"
)

// Formação de preço inexistente ia inteira para o banco e o CHECK do enum
// respondia com SQL cru em inglês ("invalid input value for enum
// price_formation_enum: \"MANUAL\" (SQLSTATE 22P02)"). Pior: a humanização de erro
// do frontend esconde texto técnico e troca por "Dados inválidos", então o usuário
// não sabia nem qual campo estava errado nem o que serve. Apareceu ao carregar a
// tabela de preço da Usimac.
func TestFormacaoDePrecoInexistenteERecusadaComOsValoresAceitos(t *testing.T) {
	_, err := NewSalesTable(1, "TABELA TESTE", PriceFormation("MANUAL"))
	if err == nil {
		t.Fatal("formação inexistente foi aceita — o CHECK do banco responderia com SQL cru")
	}
	msg := err.Error()
	for _, esperado := range []string{"MANUAL", "INFORMADO", "CUSTO_MEDIO"} {
		if !strings.Contains(msg, esperado) {
			t.Errorf("mensagem não menciona %q: %s", esperado, msg)
		}
	}

	// Controle: as formações válidas continuam passando, e o vazio vira INFORMADO —
	// senão a correção teria trocado um defeito por outro, recusando o que serve.
	for _, f := range append(FormacoesDePreco(), "") {
		st, err := NewSalesTable(1, "TABELA TESTE", PriceFormation(f))
		if err != nil {
			t.Fatalf("formação %q deveria ser aceita: %v", f, err)
		}
		if f == "" && st.PriceFormation != PriceInformado {
			t.Errorf("formação vazia virou %q, esperado INFORMADO", st.PriceFormation)
		}
	}
}
