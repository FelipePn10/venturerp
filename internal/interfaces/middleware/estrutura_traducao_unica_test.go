package middleware

import (
	"net/http/httptest"
	"testing"
)

// TestEstruturaNaoTraduzCodigoNaEntrada trava o defeito que gravava componente
// de estrutura sob o item errado em produção.
//
// O código do item era convertido duas vezes no caminho de gravação: aqui no
// middleware (comercial → interno) e de novo no caso de uso, que tenta código
// comercial primeiro. Numa base onde códigos comerciais numéricos convivem com
// chaves internas — "1", "5", "8" ao lado dos internos 1..15 — o interno 5
// produzido pelo middleware era relido como o comercial "5" e virava o interno
// 2. A gravação respondia 201 e a tela, ao recarregar o item certo, mostrava a
// estrutura vazia.
//
// Regra: na ENTRADA a estrutura não traduz; na SAÍDA traduz, porque a tela
// exibe código público.
func TestEstruturaNaoTraduzCodigoNaEntrada(t *testing.T) {
	casos := []struct {
		nome    string
		caminho string
		chave   string
		entrada bool // esperado em isItemInputReferenceKey
		saida   bool // esperado em isItemReferenceKey
	}{
		{"estrutura: pai não traduz na entrada", "/api/items/structure/create", "parent_code", false, true},
		{"estrutura: filho não traduz na entrada", "/api/items/structure/update", "child_code", false, true},
		{"estrutura: item_code segue traduzindo", "/api/items/structure/create", "item_code", true, true},
		{"fora da estrutura: pai não é referência", "/api/sales-order/create", "parent_code", false, false},
		{"outra rota: item_code traduz", "/api/production-orders", "item_code", true, true},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := httptest.NewRequest("POST", c.caminho, nil)
			if got := isItemInputReferenceKey(r, c.chave); got != c.entrada {
				t.Errorf("entrada %s em %s: esperado %v, obtido %v", c.chave, c.caminho, c.entrada, got)
			}
			if got := isItemReferenceKey(r, c.chave); got != c.saida {
				t.Errorf("saída %s em %s: esperado %v, obtido %v", c.chave, c.caminho, c.saida, got)
			}
		})
	}
}
