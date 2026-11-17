package fiscal_uc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// A consulta das NF-e recebidas com token recusado (401) dizia "tente de novo
// em instantes"; o certo é mandar conferir o token na VFIS0100.
func TestNFesRecebidas_TokenRecusadoPedeConferirToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"codigo":"nao_autorizado","mensagem":"Access token inválido"}`))
	}))
	defer srv.Close()
	t.Setenv("FOCUSNFE_BASE_URL", srv.URL)
	_, _, err := focusnfe.NewClient("x", "homologacao").ListarNFesRecebidas(context.Background(), "11222333000181", 0)
	if err == nil {
		t.Fatal("esperava erro")
	}
	msg := ErroDaEmissora(context.Background(), "a SEFAZ", "a consulta das notas recebidas", 0, err).Error()
	if !strings.Contains(msg, "token") || strings.Contains(msg, "instantes") {
		t.Fatalf("mensagem: %q", msg)
	}
}
