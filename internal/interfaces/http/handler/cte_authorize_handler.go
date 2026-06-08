package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// CTeAuthorizeHandler exposes the CT-e SEFAZ authorization endpoint.
type CTeAuthorizeHandler struct {
	authorizeUC *fiscal_uc.AuthorizeCTeUseCase
}

func NewCTeAuthorizeHandler(authorizeUC *fiscal_uc.AuthorizeCTeUseCase) *CTeAuthorizeHandler {
	return &CTeAuthorizeHandler{authorizeUC: authorizeUC}
}

func (h *CTeAuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.authorizeUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
