package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *StandardCostHandler) UpsertWorkCenterCost(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertWorkCenterCostDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpsertWorkCenterCost(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) ListWorkCenterCosts(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListWorkCenterCosts(r.Context())
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) ListWorkCenters(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(queryDefault(r, "limit", "100"))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, "limit inválido")
		return
	}
	offset, err := strconv.Atoi(queryDefault(r, "offset", "0"))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, "offset inválido")
		return
	}
	result, err := h.uc.ListWorkCenters(r.Context(), r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func queryDefault(r *http.Request, name, fallback string) string {
	if value := r.URL.Query().Get(name); value != "" {
		return value
	}
	return fallback
}

func (h *StandardCostHandler) UpsertItemPurchaseCost(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertItemPurchaseCostDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpsertItemPurchaseCost(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) GetItemPurchaseCost(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid itemCode")
		return
	}
	result, err := h.uc.GetItemPurchaseCost(r.Context(), itemCode)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) RollUp(w http.ResponseWriter, r *http.Request) {
	var dto request.CostRollupDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.RollUp(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) GetStandardCost(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid itemCode")
		return
	}
	mask := r.URL.Query().Get("mask")
	result, err := h.uc.GetStandardCost(r.Context(), itemCode, mask)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
