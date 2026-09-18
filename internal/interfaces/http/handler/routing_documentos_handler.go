package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// ─── documentos de processo ───────────────────────────────────────────────────

func (h *RoutingHandler) AddOperationDocument(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateOperationDocumentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.routeUC.AddDocument(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *RoutingHandler) UpdateOperationDocument(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "docId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código do documento inválido")
		return
	}
	var dto request.UpdateOperationDocumentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.routeUC.UpdateDocument(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) RemoveOperationDocument(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "docId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código do documento inválido")
		return
	}
	if err := h.routeUC.RemoveDocument(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListOperationDocuments devolve os documentos da operação de biblioteca.
func (h *RoutingHandler) ListOperationDocuments(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código da operação inválido")
		return
	}
	result, err := h.routeUC.ListDocumentsByOperation(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ListStepDocuments devolve o que o operador vê na etapa: os documentos dela
// MAIS os que a operação de biblioteca carrega.
func (h *RoutingHandler) ListStepDocuments(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código da etapa inválido")
		return
	}
	result, err := h.routeUC.ListDocumentsForStep(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── pontos de inspeção ───────────────────────────────────────────────────────

func (h *RoutingHandler) AddRouteInspection(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código da etapa inválido")
		return
	}
	var dto request.CreateRouteInspectionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.RouteOperationID = opID
	result, err := h.routeUC.AddInspection(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *RoutingHandler) RemoveRouteInspection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "inspectionId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código da inspeção inválido")
		return
	}
	if err := h.routeUC.RemoveInspection(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
