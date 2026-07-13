package handler

import (
	"encoding/json"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_tolerance_uc"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

type PurchaseToleranceHandler struct {
	uc *purchase_tolerance_uc.UseCase
}

func NewPurchaseToleranceHandler(uc *purchase_tolerance_uc.UseCase) *PurchaseToleranceHandler {
	return &PurchaseToleranceHandler{uc: uc}
}
func (h *PurchaseToleranceHandler) Save(w http.ResponseWriter, r *http.Request) {
	var d request.UpsertPurchaseToleranceDTO
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	x, err := h.uc.Save(r.Context(), d)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	status := http.StatusCreated
	if d.ID > 0 {
		status = http.StatusOK
	}
	jsonResponse(w, status, x)
}
func (h *PurchaseToleranceHandler) List(w http.ResponseWriter, r *http.Request) {
	var supplier *int64
	if raw := r.URL.Query().Get("supplier_code"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid supplier_code")
			return
		}
		supplier = &v
	}
	x, err := h.uc.List(r.Context(), supplier)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, x)
}
func (h *PurchaseToleranceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err = h.uc.Delete(r.Context(), id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *PurchaseToleranceHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var d request.EvaluatePurchaseToleranceDTO
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	x, err := h.uc.Evaluate(r.Context(), d)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, x)
}
