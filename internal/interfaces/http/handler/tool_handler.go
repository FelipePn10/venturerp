package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/tool_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ToolHandler) UpdateTool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de ferramenta inválido")
		return
	}
	var dto request.UpdateToolDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.uc.Update(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) GetTool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de ferramenta inválido")
		return
	}
	result, err := h.uc.Get(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") == "true"
	result, err := h.uc.List(r.Context(), onlyActive)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) DeactivateTool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de ferramenta inválido")
		return
	}
	if err := h.uc.Deactivate(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ToolHandler) ResetToolLife(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de ferramenta inválido")
		return
	}
	result, err := h.uc.ResetLife(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) ListToolsNeedingReplacement(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListNeedingReplacement(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── serials (physical instances of a tool master) ────────────────────────────

func (h *ToolHandler) CreateSerial(w http.ResponseWriter, r *http.Request) {
	toolID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de ferramenta inválido")
		return
	}
	var dto request.CreateToolSerialDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.ToolID = toolID
	if dto.CreatedBy == uuid.Nil {
		dto.CreatedBy = actingUser(r)
	}
	result, err := h.uc.CreateSerial(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ToolHandler) ListSerials(w http.ResponseWriter, r *http.Request) {
	toolID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de ferramenta inválido")
		return
	}
	onlyActive := r.URL.Query().Get("only_active") == "true"
	result, err := h.uc.ListSerials(r.Context(), toolID, onlyActive)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) GetSerial(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "serialId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de série inválido")
		return
	}
	result, err := h.uc.GetSerial(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) UpdateSerial(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "serialId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de série inválido")
		return
	}
	var dto request.UpdateToolSerialDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.uc.UpdateSerial(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) DeactivateSerial(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "serialId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador de série inválido")
		return
	}
	if err := h.uc.DeactivateSerial(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── association with route operations ─────────────────────────────────────────

func (h *ToolHandler) AddRouteOpTool(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código da operação inválido")
		return
	}
	var dto request.AddRouteOpToolDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.RouteOperationID = opID
	result, err := h.uc.AddToOperation(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ToolHandler) ListRouteOpTools(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código da operação inválido")
		return
	}
	result, err := h.uc.ListByOperation(r.Context(), opID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ToolHandler) RemoveRouteOpTool(w http.ResponseWriter, r *http.Request) {
	linkID, err := strconv.ParseInt(chi.URLParam(r, "toolLinkId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "vínculo de ferramenta inválido")
		return
	}
	if err := h.uc.RemoveFromOperation(r.Context(), linkID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
