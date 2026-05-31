package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc"
	"github.com/go-chi/chi/v5"
)

type PurchaseRequisitionHandler struct {
	uc       *purchase_requisition_uc.PurchaseRequisitionUseCase
	generate *purchase_requisition_uc.GeneratePurchaseOrdersUseCase
}

func NewPurchaseRequisitionHandler(
	uc *purchase_requisition_uc.PurchaseRequisitionUseCase,
	generate *purchase_requisition_uc.GeneratePurchaseOrdersUseCase,
) *PurchaseRequisitionHandler {
	return &PurchaseRequisitionHandler{uc: uc, generate: generate}
}

func (h *PurchaseRequisitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePurchaseRequisitionDTO
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

func (h *PurchaseRequisitionHandler) Get(w http.ResponseWriter, r *http.Request) {
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

func (h *PurchaseRequisitionHandler) List(w http.ResponseWriter, r *http.Request) {
	onlyOpen := r.URL.Query().Get("only_open") == "true"
	res, err := h.uc.List(r.Context(), onlyOpen)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchaseRequisitionHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.AddRequisitionItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.RequisitionCode = code
	res, err := h.uc.AddItem(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

// GeneratePurchaseOrders turns selected requisition items into purchase orders.
func (h *PurchaseRequisitionHandler) GeneratePurchaseOrders(w http.ResponseWriter, r *http.Request) {
	var dto request.GeneratePurchaseOrdersDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.generate.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}
