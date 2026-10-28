package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
	"github.com/go-chi/chi/v5"
)

// Id fora do formato UUID é erro de quem chamou (400), não erro interno: a
// exploração das telas mandou "0" e "10" e recebia 500.
func TestPasswordChangeIDInvalidoResponde400(t *testing.T) {
	h := NewPasswordChangeHandler(nil)
	casos := []struct {
		nome string
		fn   http.HandlerFunc
		body string
	}{
		{"aprovar", h.Approve, ""},
		{"rejeitar", h.Reject, `{"reason":"x"}`},
		{"concluir", h.Complete, `{"current_password":"a","new_password":"b","confirm_password":"b"}`},
	}
	for _, c := range casos {
		for _, id := range []string{"0", "10", "abc"} {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.body))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("requestID", id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			c.fn(rec, req)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "identificador") {
				t.Fatalf("%s com id %q: %d %s", c.nome, id, rec.Code, rec.Body.String())
			}
		}
	}
}

// Senha atual errada não é sessão inválida: responder 401 derrubava a sessão
// no app.
func TestPasswordChangeSenhaAtualErradaNaoResponde401(t *testing.T) {
	h := NewPasswordChangeHandler(nil)
	rec := httptest.NewRecorder()
	h.writeError(rec, httptest.NewRequest(http.MethodPost, "/", nil), user_uc.ErrCurrentPasswordInvalid)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "senha atual") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
