package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/preferencias_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// MinhasPreferencias: GET e PUT /api/acesso/minhas-preferencias — as
// preferências de exibição do usuário da sessão.
func MinhasPreferencias(uc *preferencias_uc.UseCase) (get, put http.HandlerFunc) {
	get = func(w http.ResponseWriter, r *http.Request) {
		p, err := uc.Get(r.Context())
		customerRespond(w, p, err)
	}
	put = func(w http.ResponseWriter, r *http.Request) {
		var p preferencias_uc.Preferencias
		if !security.DecodeBody(w, r, &p) {
			return
		}
		saved, err := uc.Save(r.Context(), p)
		customerRespond(w, saved, err)
	}
	return get, put
}
