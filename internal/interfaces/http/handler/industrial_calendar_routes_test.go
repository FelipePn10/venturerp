package handler

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestIndustrialCalendarRoutesPublishesGenerationAndLegacyContracts(t *testing.T) {
	h := &IndustrialCalendarHandler{}
	got := map[string]bool{}
	if err := chi.Walk(h.Routes(), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		got[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("percorrendo rotas: %v", err)
	}
	want := []string{
		"POST /create", "POST /generate", "POST /generate/{year}/{month}",
		"GET /month/{year}/{month}", "GET /workdays/{year}/{month}",
	}
	for _, route := range want {
		if !got[route] {
			t.Errorf("rota obrigatória ausente: %s (publicadas: %v)", route, got)
		}
	}
}
