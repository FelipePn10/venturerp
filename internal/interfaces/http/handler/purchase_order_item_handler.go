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

// PurchaseOrderItemHandler mantém as linhas do pedido de compra: inclui (com
// preço / UM interna / IPI resolvidos), altera e cancela ou elimina o saldo.
type PurchaseOrderItemHandler struct {
	addUC    *purchase_order_uc.AddPurchaseOrderItemUseCase
	updateUC *purchase_order_uc.UpdatePurchaseOrderItemUseCase
	cancelUC *purchase_order_uc.CancelPurchaseOrderItemUseCase
}

func NewPurchaseOrderItemHandler(addUC *purchase_order_uc.AddPurchaseOrderItemUseCase, updateUC *purchase_order_uc.UpdatePurchaseOrderItemUseCase, cancelUC *purchase_order_uc.CancelPurchaseOrderItemUseCase) *PurchaseOrderItemHandler {
	return &PurchaseOrderItemHandler{addUC: addUC, updateUC: updateUC, cancelUC: cancelUC}
}

func (h *PurchaseOrderItemHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto request.CreatePurchaseOrderItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.PurchaseOrderCode = code
	res, err := h.addUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, res)
}

func pedidoELinha(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do pedido inválido")
		return 0, 0, false
	}
	linha, err := strconv.ParseInt(chi.URLParam(r, "lineCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código da linha inválido")
		return 0, 0, false
	}
	return code, linha, true
}

// UpdateItem: PUT /api/purchase-order/{code}/items/{lineCode}
func (h *PurchaseOrderItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	code, linha, ok := pedidoELinha(w, r)
	if !ok {
		return
	}
	var dto request.UpdatePurchaseOrderItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.PurchaseOrderCode, dto.ItemLineCode = code, linha
	res, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, res)
}

// CancelItem: POST /api/purchase-order/{code}/items/{lineCode}/cancel
func (h *PurchaseOrderItemHandler) CancelItem(w http.ResponseWriter, r *http.Request) {
	code, linha, ok := pedidoELinha(w, r)
	if !ok {
		return
	}
	var dto request.CancelPurchaseOrderItemDTO
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	dto.PurchaseOrderCode, dto.ItemLineCode = code, linha
	res, err := h.cancelUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, res)
}
