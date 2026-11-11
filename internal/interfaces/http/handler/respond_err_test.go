package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Erro simples (sem tipo) é a validação que muitos casos de uso ainda devolvem:
// sai com o status do handler e a mensagem. Antes respondErr e respondErrStatus
// se chamavam sem fim e o estouro de pilha derrubava a API.
func TestRespondErrComErroSimplesNaoDerrubaAAPI(t *testing.T) {
	for _, c := range []struct {
		responder func(http.ResponseWriter)
		status    int
	}{
		{func(w http.ResponseWriter) { respondErr(w, errors.New("informe o item")) }, http.StatusUnprocessableEntity},
		{func(w http.ResponseWriter) {
			respondErrStatus(w, http.StatusNotFound, errors.New("plano não encontrado"))
		}, http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		c.responder(rec)
		if rec.Code != c.status {
			t.Fatalf("status %d, esperado %d", rec.Code, c.status)
		}
		if !strings.Contains(rec.Body.String(), "informe o item") && !strings.Contains(rec.Body.String(), "plano não encontrado") {
			t.Fatalf("a mensagem do erro sumiu: %s", rec.Body.String())
		}
	}
}
