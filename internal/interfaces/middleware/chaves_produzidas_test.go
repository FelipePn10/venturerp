package middleware

import (
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
