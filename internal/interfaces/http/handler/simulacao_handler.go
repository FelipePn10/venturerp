package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/system_update_uc"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// SimulacaoHandler: o modo simulação (cópia do banco da empresa para o treino).
type SimulacaoHandler struct {
	M *system_update_uc.SimulacaoManager
}

// Pedir: POST /api/system/simulacao — pede a cópia (ADMIN).
func (h *SimulacaoHandler) Pedir(w http.ResponseWriter, r *http.Request) {
	quem := ""
	if u, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser); ok && u != nil {
		quem = u.ID
	}
	st, err := h.M.Pedir(r.Context(), quem)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, st)
}

// Status: GET /api/system/simulacao/status.
func (h *SimulacaoHandler) Status(w http.ResponseWriter, r *http.Request) {
	st, err := h.M.Status(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, st)
}
