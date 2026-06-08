package handler

import (
	"net/http"
	"strconv"

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
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	report, err := h.uc.Execute(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, report)
}
