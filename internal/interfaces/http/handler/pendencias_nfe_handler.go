package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// PendenciasNFeHandler expõe as pendências de efeitos da NF-e (estoque,
// títulos e pedido que ainda não entraram) e o reprocesso manual.
type PendenciasNFeHandler struct {
	efeitos *fiscal_uc.EfeitosNFe
	auth    ports.AuthService
}

func NewPendenciasNFeHandler(efeitos *fiscal_uc.EfeitosNFe, auth ports.AuthService) *PendenciasNFeHandler {
	return &PendenciasNFeHandler{efeitos: efeitos, auth: auth}
}

func (h *PendenciasNFeHandler) Listar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	lista, err := h.efeitos.DaNota(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, lista)
}

func (h *PendenciasNFeHandler) Reprocessar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	userID, err := h.auth.UserID(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	lista, err := h.efeitos.Reprocessar(r.Context(), id, userID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, lista)
}
