package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/pendencias_uc"
)

// PendenciasHandler expõe a central de pendências do dia.
type PendenciasHandler struct{ Central *pendencias_uc.CentralUseCase }

// DoDia: GET /api/pendencias/dia — o que o perfil do usuário tem a resolver hoje.
func (h *PendenciasHandler) DoDia(w http.ResponseWriter, r *http.Request) {
	out, err := h.Central.DoDia(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}
