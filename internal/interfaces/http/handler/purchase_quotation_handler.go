package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_quotation_uc"
	"github.com/go-chi/chi/v5"
)

type PurchaseQuotationHandler struct {
	uc       *purchase_quotation_uc.PurchaseQuotationUseCase
	generate *purchase_quotation_uc.GenerateOrdersFromQuotationUseCase
}

func NewPurchaseQuotationHandler(
	uc *purchase_quotation_uc.PurchaseQuotationUseCase,
	generate *purchase_quotation_uc.GenerateOrdersFromQuotationUseCase,
) *PurchaseQuotationHandler {
	return &PurchaseQuotationHandler{uc: uc, generate: generate}
}

func (h *PurchaseQuotationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePurchaseQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchaseQuotationHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.Get(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchaseQuotationHandler) List(w http.ResponseWriter, r *http.Request) {
	onlyOpen := r.URL.Query().Get("only_open") == "true"
	res, err := h.uc.List(r.Context(), onlyOpen)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchaseQuotationHandler) AddSupplier(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.AddQuotationSupplierDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.QuotationCode = code
	res, err := h.uc.AddSupplier(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchaseQuotationHandler) RecordPrice(w http.ResponseWriter, r *http.Request) {
	var dto request.RecordQuotationPriceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.RecordPrice(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchaseQuotationHandler) SelectPrice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "priceID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid price id")
		return
	}
	res, err := h.uc.SelectPrice(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchaseQuotationHandler) GenerateOrders(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.GenerateOrdersFromQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.QuotationCode = code
	res, err := h.generate.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}
