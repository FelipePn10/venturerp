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
