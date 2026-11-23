package middleware

import "testing"

func TestCaminhoSemSegredo(t *testing.T) {
	casos := map[string]string{
		"/api/portal/abc123":                         "/api/portal/***",
		"/api/portal/abc123/pedidos/9/repetir":       "/api/portal/***/pedidos/9/repetir",
		"/api/portal/abc/notas/1/danfe":              "/api/portal/***/notas/1/danfe",
		"/api/integracoes/inter/webhook/segredo-xyz": "/api/integracoes/inter/webhook/***",
		"/api/customers/24/portal-links":             "/api/customers/24/portal-links",
		"/api/integracoes/inter/webhook-registrar":   "/api/integracoes/inter/webhook-registrar",
	}
	for in, want := range casos {
		if got := CaminhoSemSegredo(in); got != want {
			t.Errorf("%s: %s, esperado %s", in, got, want)
		}
	}
}
