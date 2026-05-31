package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc"
	"github.com/go-chi/chi/v5"
)

type PurchasePriceHandler struct {
	uc *purchase_price_uc.PurchasePriceUseCase
}

func NewPurchasePriceHandler(uc *purchase_price_uc.PurchasePriceUseCase) *PurchasePriceHandler {
	return &PurchasePriceHandler{uc: uc}
}

func (h *PurchasePriceHandler) CreateTable(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePurchasePriceTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.CreateTable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchasePriceHandler) UpdateTable(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdatePurchasePriceTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.UpdateTable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.GetTable(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	res, err := h.uc.ListTables(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	var dto request.AddPurchasePriceItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.AddItem(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchasePriceHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.ListItems(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeleteItem(r.Context(), id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
