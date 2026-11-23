package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCodigosDoAvisoInter(t *testing.T) {
	for corpo, want := range map[string][]string{
		`[{"codigoSolicitacao":"a"},{"codigoSolicitacao":"b"},{"codigoSolicitacao":"a"}]`: {"a", "b"},
		`{"payload":[{"codigoSolicitacao":"c"}]}`:                                         {"c"},
		`{"codigoSolicitacao":"d"}`:                                                       {"d"},
		`lixo`:                                                                            nil,
	} {
		if got := codigosDoAviso([]byte(corpo)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s → %v, quer %v", corpo, got, want)
		}
	}
}

// Sem o segredo certo no caminho, 404 (e nada é processado).
func TestWebhookInterExigeOSegredo(t *testing.T) {
	h := &InterWebhookHandler{Token: "segredo-longo"}
	r := chi.NewRouter()
	r.Post("/api/integracoes/inter/webhook/{token}", h.Receber)
	for tok, want := range map[string]int{"errado": 404, "segredo-longo": 200} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/integracoes/inter/webhook/"+tok, strings.NewReader(`[]`)))
		if rec.Code != want {
			t.Errorf("token %s: %d, quer %d", tok, rec.Code, want)
		}
	}
}
