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

// PurchaseOrderItemHandler adds items to a purchase order, with automatic
// resolution of price / internal UM / IPI.
type PurchaseOrderItemHandler struct {
	addUC *purchase_order_uc.AddPurchaseOrderItemUseCase
}

func NewPurchaseOrderItemHandler(addUC *purchase_order_uc.AddPurchaseOrderItemUseCase) *PurchaseOrderItemHandler {
	return &PurchaseOrderItemHandler{addUC: addUC}
}

func (h *PurchaseOrderItemHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
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
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, res)
}
