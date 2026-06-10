package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_movement_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
	"github.com/go-chi/chi/v5"
)

type StockMovementTypeHandler struct {
	uc *stock_movement_uc.StockMovementTypeUseCase
}

func NewStockMovementTypeHandler(uc *stock_movement_uc.StockMovementTypeUseCase) *StockMovementTypeHandler {
	return &StockMovementTypeHandler{uc: uc}
}

func (h *StockMovementTypeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var s entity.StockMovementType
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), &s)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *StockMovementTypeHandler) Update(w http.ResponseWriter, r *http.Request) {
	var s entity.StockMovementType
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.Update(r.Context(), &s)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StockMovementTypeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StockMovementTypeHandler) GetBySigla(w http.ResponseWriter, r *http.Request) {
	sigla := chi.URLParam(r, "sigla")
	result, err := h.uc.GetBySigla(r.Context(), sigla)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StockMovementTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
