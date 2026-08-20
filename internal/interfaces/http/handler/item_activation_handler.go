package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/go-chi/chi/v5"
)

// ItemActivationHandler exposes the engineering readiness validation for an item.
type ItemActivationHandler struct {
	uc *item_uc.ValidateItemActivationUseCase
}

func NewItemActivationHandler(uc *item_uc.ValidateItemActivationUseCase) *ItemActivationHandler {
	return &ItemActivationHandler{uc: uc}
}

// ValidateActivation returns the cross-validation report (BOM/routing/supplier/UOM)
// telling whether the item is ready to take part in the MRP/production/purchasing flow.
func (h *ItemActivationHandler) ValidateActivation(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	report, err := h.uc.ExecuteBusinessCode(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, report)
}
