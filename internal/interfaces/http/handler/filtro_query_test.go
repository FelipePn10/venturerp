package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Filtro mal digitado responde 422 com o motivo, em vez de listar tudo.
func TestFiltroValido(t *testing.T) {
	for _, c := range []struct {
		query  string
		ok     bool
		trecho string
	}{
		{"from=2026-10-01&to=2026-10-31&customer_code=12", true, ""},
		{"", true, ""},
		{"from=31/02/2026", false, "data inicial"},
		{"to=amanha", false, "data final"},
		{"customer_code=abc", false, "customer_code"},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/x?"+c.query, nil)
		if got := filtroValido(w, r, "customer_code"); got != c.ok {
			t.Fatalf("%q: ok=%v, esperado %v", c.query, got, c.ok)
		}
		if !c.ok && (w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), c.trecho)) {
			t.Fatalf("%q: %d %s", c.query, w.Code, w.Body.String())
		}
	}
}

func TestQueryValida(t *testing.T) {
	f := filtrosDaQuery{Inteiros: []string{"customer_code"}, Decimais: []string{"qty"}, Listas: []string{"codes"}, Datas: []string{"from"}}
	casos := []struct {
		query  string
		valido bool
	}{
		{"", true},
		{"customer_code=12&qty=1.5&codes=1,2&from=2026-10-14", true},
		{"customer_code=undefined&qty=null&codes=&from=", true}, // filtro em branco vindo da tela
		{"from=14/10/2026", true},
		{"customer_code=abc", false},
		{"qty=dez", false},
		{"codes=1,x", false},
		{"from=2026-02-30", false},
	}
	for _, c := range casos {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/x?"+c.query, nil)
		if got := queryValida(w, r, f); got != c.valido {
			t.Errorf("%q: válido=%v, quer %v (%s)", c.query, got, c.valido, w.Body.String())
		}
		if !c.valido && w.Code != http.StatusUnprocessableEntity {
			t.Errorf("%q: status %d, quer 422", c.query, w.Code)
		}
	}
}
