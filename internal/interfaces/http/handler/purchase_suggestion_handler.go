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

// PurchaseSuggestionHandler exposes the MRP purchase-suggestion workflow:
// list PURCHASE planned orders that are suggestions, approve one into a purchase
// order, or reject it.
type PurchaseSuggestionHandler struct {
	listUC    *purchase_order_uc.ListPurchaseSuggestionsUseCase
	approveUC *purchase_order_uc.ApprovePurchaseSuggestionUseCase
	rejectUC  *purchase_order_uc.RejectPurchaseSuggestionUseCase
}

func NewPurchaseSuggestionHandler(
	listUC *purchase_order_uc.ListPurchaseSuggestionsUseCase,
	approveUC *purchase_order_uc.ApprovePurchaseSuggestionUseCase,
	rejectUC *purchase_order_uc.RejectPurchaseSuggestionUseCase,
) *PurchaseSuggestionHandler {
	return &PurchaseSuggestionHandler{listUC: listUC, approveUC: approveUC, rejectUC: rejectUC}
}

func (h *PurchaseSuggestionHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseSuggestionHandler) Approve(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.ApprovePurchaseSuggestionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.PlannedOrderCode = code
	result, err := h.approveUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *PurchaseSuggestionHandler) Reject(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.rejectUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
