package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc"
	"github.com/go-chi/chi/v5"
)

// SupplierSefazHandler exposes the SEFAZ cadastral query for a supplier.
type SupplierSefazHandler struct {
	uc *supplier_uc.ConsultSupplierSefazUseCase
}

func NewSupplierSefazHandler(uc *supplier_uc.ConsultSupplierSefazUseCase) *SupplierSefazHandler {
	return &SupplierSefazHandler{uc: uc}
}

func (h *SupplierSefazHandler) Query(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.Execute(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}
