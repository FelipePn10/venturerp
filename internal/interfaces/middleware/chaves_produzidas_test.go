package middleware

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestChavesProduzidasPelaTraducaoDoCorpo(t *testing.T) {
	casos := []struct {
		valor any
		quer  []string
	}{
		{int64(5), []string{"5"}},
		{float64(5), []string{"5"}}, // contrato legado: número JSON
		{"12", []string{"12"}},      // preço: a chave volta como texto
		{[]any{int64(5), "7"}, []string{"5", "7"}},
		{float64(1.5), nil},
		{"RN-01001", nil},
		{nil, nil},
	}
	for _, c := range casos {
		if got := chavesProduzidas(c.valor); !reflect.DeepEqual(got, c.quer) {
			t.Fatalf("chavesProduzidas(%#v) = %#v, quer %#v", c.valor, got, c.quer)
		}
	}
}

// Linha de documento não é item: traduzir esses campos trocava o número da
// linha pelo código comercial do item que tivesse a mesma chave.
func TestLinhaDeDocumentoNaoEhItem(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/production-plan/list", nil)
	for _, campo := range []string{"order_item_code", "reference_item_code"} {
		if isItemReferenceKey(r, campo) {
			t.Fatalf("%s é linha de documento e não pode ser traduzido como item", campo)
		}
	}
	for _, campo := range []string{"item_code", "item_codes", "child_item_code", "service_item_code"} {
		if !isItemReferenceKey(r, campo) {
			t.Fatalf("%s é referência de item e precisa continuar traduzido", campo)
		}
	}
}
