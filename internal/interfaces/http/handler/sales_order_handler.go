package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type SalesOrderHandler struct {
	createUC         *sales_order_uc.CreateSalesOrderUseCase
	updateUC         *sales_order_uc.UpdateSalesOrderUseCase
	getUC            *sales_order_uc.GetSalesOrderUseCase
	listUC           *sales_order_uc.ListSalesOrdersUseCase
	listByCustomerUC *sales_order_uc.ListSalesOrdersByCustomerUseCase
	listByStatusUC   *sales_order_uc.ListSalesOrdersByStatusUseCase
	cancelUC         *sales_order_uc.CancelSalesOrderUseCase
	blockUC          *sales_order_uc.BlockSalesOrderUseCase
	unblockUC        *sales_order_uc.UnblockSalesOrderUseCase
	changeStatusUC   *sales_order_uc.ChangeStatusSalesOrderUseCase

	createItemUC *sales_order_uc.CreateSalesOrderItemUseCase
	updateItemUC *sales_order_uc.UpdateSalesOrderItemUseCase
	listItemsUC  *sales_order_uc.ListSalesOrderItemsUseCase
	cancelItemUC *sales_order_uc.CancelSalesOrderItemUseCase
}

func (h *SalesOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesOrderDTO
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

func (h *SalesOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.UpdateSalesOrderDTO
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

func (h *SalesOrderHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
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

func (h *SalesOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) ListByCustomer(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "customerCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid customer code")
		return
	}
	results, err := h.listByCustomerUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	results, err := h.listByStatusUC.Execute(r.Context(), status)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
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

func (h *SalesOrderHandler) Block(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.BlockSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.blockUC.Execute(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.unblockUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.ChangeStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.changeStatusUC.Execute(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Item handlers

func (h *SalesOrderHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesOrderItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createItemUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesOrderHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	var dto request.UpdateSalesOrderItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.updateItemUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesOrderHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	results, err := h.listItemsUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) CancelItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	if err := h.cancelItemUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
