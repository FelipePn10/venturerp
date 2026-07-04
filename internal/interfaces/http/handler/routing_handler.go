package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

// ─── operations ──────────────────────────────────────────────────────────────

func (h *RoutingHandler) CreateOperation(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.operationUC.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *RoutingHandler) UpdateOperation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.operationUC.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) GetOperation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.operationUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) ListOperations(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.operationUC.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) DeactivateOperation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.operationUC.Deactivate(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── routes ──────────────────────────────────────────────────────────────────

func (h *RoutingHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRouteDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.routeUC.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *RoutingHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateRouteDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.routeUC.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) GetRouteDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.routeUC.GetDetail(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) ListRoutesByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(r.URL.Query().Get("item_code"), 10, 64)
	if err != nil || itemCode <= 0 {
		jsonError(w, http.StatusBadRequest, "item_code must be a positive integer")
		return
	}
	result, err := h.routeUC.ListByItem(r.Context(), itemCode)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) DeactivateRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.routeUC.Deactivate(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── route operations ─────────────────────────────────────────────────────────

func (h *RoutingHandler) AddRouteOperation(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "routeId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid routeId")
		return
	}
	var dto request.AddRouteOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.RouteID = routeID
	result, err := h.routeUC.AddOperation(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *RoutingHandler) UpdateRouteOperation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid opId")
		return
	}
	var dto request.UpdateRouteOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.routeUC.UpdateOperation(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) RemoveRouteOperation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid opId")
		return
	}
	if err := h.routeUC.RemoveOperation(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── network ─────────────────────────────────────────────────────────────────

func (h *RoutingHandler) GetNetworkEdges(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid route id")
		return
	}
	edges, err := h.routeUC.GetEdges(r.Context(), routeID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, edges)
}

func (h *RoutingHandler) SetNetworkEdge(w http.ResponseWriter, r *http.Request) {
	var dto request.SetNetworkEdgeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.routeUC.SetEdge(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) DeleteNetworkEdge(w http.ResponseWriter, r *http.Request) {
	var dto request.DeleteNetworkEdgeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.routeUC.DeleteEdge(r.Context(), dto); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── alternative resources ────────────────────────────────────────────────────

func (h *RoutingHandler) AddRouteOpResource(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid opId")
		return
	}
	var dto request.AddRouteOpResourceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.RouteOperationID = opID
	result, err := h.routeUC.AddResource(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *RoutingHandler) ListRouteOpResources(w http.ResponseWriter, r *http.Request) {
	opID, err := strconv.ParseInt(chi.URLParam(r, "opId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid opId")
		return
	}
	result, err := h.routeUC.ListResources(r.Context(), opID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) UpdateRouteOpResource(w http.ResponseWriter, r *http.Request) {
	resID, err := strconv.ParseInt(chi.URLParam(r, "resourceId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid resourceId")
		return
	}
	var dto request.UpdateRouteOpResourceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = resID
	result, err := h.routeUC.UpdateResource(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) SetRouteOpResourcePrimary(w http.ResponseWriter, r *http.Request) {
	resID, err := strconv.ParseInt(chi.URLParam(r, "resourceId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid resourceId")
		return
	}
	result, err := h.routeUC.SetPrimaryResource(r.Context(), resID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) RemoveRouteOpResource(w http.ResponseWriter, r *http.Request) {
	resID, err := strconv.ParseInt(chi.URLParam(r, "resourceId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid resourceId")
		return
	}
	if err := h.routeUC.RemoveResource(r.Context(), resID); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── lead time ───────────────────────────────────────────────────────────────

func (h *RoutingHandler) GetLeadTime(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid route id")
		return
	}
	// Optional ?qty= scales the run (per-piece) portion of the lead time; defaults to 1.
	qty := 1.0
	if q := r.URL.Query().Get("qty"); q != "" {
		if parsed, perr := strconv.ParseFloat(q, 64); perr == nil && parsed > 0 {
			qty = parsed
		}
	}
	result, err := h.leadTimeUC.Execute(r.Context(), id, qty)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
