package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/tool_uc"
	"github.com/go-chi/chi/v5"
)

type ToolHandler struct {
	uc *tool_uc.ToolUseCase
}

func NewToolHandler(uc *tool_uc.ToolUseCase) *ToolHandler {
	return &ToolHandler{uc: uc}
}

// ─── master ────────────────────────────────────────────────────────────────────

func (h *ToolHandler) CreateTool(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateToolDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ToolHandler) UpdateTool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid tool id")
		return
	}
	var dto request.UpdateToolDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.uc.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) GetTool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid tool id")
		return
	}
	result, err := h.uc.Get(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") == "true"
	result, err := h.uc.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) DeactivateTool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid tool id")
		return
	}
	if err := h.uc.Deactivate(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ToolHandler) ResetToolLife(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid tool id")
		return
	}
	result, err := h.uc.ResetLife(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) ListToolsNeedingReplacement(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListNeedingReplacement(r.Context())
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── association with route operations ─────────────────────────────────────────

func (h *ToolHandler) AddRouteOpTool(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid opId")
		return
	}
	var dto request.AddRouteOpToolDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.RouteOperationID = opID
	result, err := h.uc.AddToOperation(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ToolHandler) ListRouteOpTools(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid opId")
		return
	}
	result, err := h.uc.ListByOperation(r.Context(), opID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) RemoveRouteOpTool(w http.ResponseWriter, r *http.Request) {
	linkID, err := strconv.ParseInt(chi.URLParam(r, "toolLinkId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid toolLinkId")
		return
	}
	if err := h.uc.RemoveFromOperation(r.Context(), linkID); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
