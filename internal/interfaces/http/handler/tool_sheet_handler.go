package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/tool_sheet_uc"
	"github.com/go-chi/chi/v5"
)

// ToolSheetHandler serves the "Ficha de Produção da Ferramenta" endpoints.
type ToolSheetHandler struct {
	uc *tool_sheet_uc.ToolSheetUseCase
}

func NewToolSheetHandler(uc *tool_sheet_uc.ToolSheetUseCase) *ToolSheetHandler {
	return &ToolSheetHandler{uc: uc}
}

// ListOrders returns the production orders eligible for the sheet (OFC excluded).
// Optional query param `q` filters by order number / item code / item name.
func (h *ToolSheetHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListOrders(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// GetSheet returns the full sheet (header + operations + serials) for an order.
// Doubles as the "Atualiza" refresh.
func (h *ToolSheetHandler) GetSheet(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "orderId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	result, err := h.uc.GetSheet(r.Context(), orderID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// Assign binds a serial to an operation/tool.
func (h *ToolSheetHandler) Assign(w http.ResponseWriter, r *http.Request) {
	var dto request.AssignToolSerialDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.AssignedBy = actingUser(r)
	result, err := h.uc.Assign(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// Substitute replaces the serial already bound to an operation/tool.
func (h *ToolSheetHandler) Substitute(w http.ResponseWriter, r *http.Request) {
	var dto request.SubstituteToolSerialDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.SubstitutedBy = actingUser(r)
	result, err := h.uc.Substitute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ListSubstitutions returns the substitution history for an operation/tool.
func (h *ToolSheetHandler) ListSubstitutions(w http.ResponseWriter, r *http.Request) {
	operationID, err := strconv.ParseInt(r.URL.Query().Get("operation_id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid operation_id")
		return
	}
	toolID, err := strconv.ParseInt(r.URL.Query().Get("tool_id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid tool_id")
		return
	}
	result, err := h.uc.ListSubstitutions(r.Context(), operationID, toolID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
