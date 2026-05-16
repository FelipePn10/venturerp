package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type PurchaseOrderHandler struct {
	createUC          *purchase_order_uc.CreatePurchaseOrderUseCase
	updateUC          *purchase_order_uc.UpdatePurchaseOrderUseCase
	getUC             *purchase_order_uc.GetPurchaseOrderUseCase
	listUC            *purchase_order_uc.ListPurchaseOrdersUseCase
	listBySupplierUC  *purchase_order_uc.ListPurchaseOrdersBySupplierUseCase
	listByStatusUC    *purchase_order_uc.ListPurchaseOrdersByStatusUseCase
	cancelUC          *purchase_order_uc.CancelPurchaseOrderUseCase
}

func NewPurchaseOrderHandler(
	createUC *purchase_order_uc.CreatePurchaseOrderUseCase,
	updateUC *purchase_order_uc.UpdatePurchaseOrderUseCase,
	getUC *purchase_order_uc.GetPurchaseOrderUseCase,
	listUC *purchase_order_uc.ListPurchaseOrdersUseCase,
	listBySupplierUC *purchase_order_uc.ListPurchaseOrdersBySupplierUseCase,
	listByStatusUC *purchase_order_uc.ListPurchaseOrdersByStatusUseCase,
	cancelUC *purchase_order_uc.CancelPurchaseOrderUseCase,
) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{
		createUC:         createUC,
		updateUC:         updateUC,
		getUC:            getUC,
		listUC:           listUC,
		listBySupplierUC: listBySupplierUC,
		listByStatusUC:   listByStatusUC,
		cancelUC:         cancelUC,
	}
}

func (h *PurchaseOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePurchaseOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *PurchaseOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.UpdatePurchaseOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PurchaseOrderHandler) ListBySupplier(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "supplierCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid supplier code")
		return
	}
	results, err := h.listBySupplierUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PurchaseOrderHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	results, err := h.listByStatusUC.Execute(r.Context(), status)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PurchaseOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.cancelUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
