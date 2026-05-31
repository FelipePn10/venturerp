package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc"
	"github.com/go-chi/chi/v5"
)

type ItemSupplierHandler struct {
	uc *item_supplier_uc.ItemSupplierUseCase
}

func NewItemSupplierHandler(uc *item_supplier_uc.ItemSupplierUseCase) *ItemSupplierHandler {
	return &ItemSupplierHandler{uc: uc}
}

func (h *ItemSupplierHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertItemPreferredSupplierDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.Upsert(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ItemSupplierHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	res, err := h.uc.ListByItem(r.Context(), itemCode)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemSupplierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.Delete(r.Context(), id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
