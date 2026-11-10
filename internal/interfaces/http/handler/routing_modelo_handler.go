package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// ListRouteTemplates lista os roteiros-modelo (?incluir_inativos=true traz os desativados).
func (h *RoutingHandler) ListRouteTemplates(w http.ResponseWriter, r *http.Request) {
	result, err := h.routeUC.ListTemplates(r.Context(), r.URL.Query().Get("incluir_inativos") == "true")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// CopyRoute cria um roteiro a partir do roteiro {id} (modelo → item, item →
// item ou item → modelo), numa transação só.
func (h *RoutingHandler) CopyRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código do roteiro inválido")
		return
	}
	var dto request.CopyRouteDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	dto.SourceRouteID = id
	result, err := h.routeUC.Copy(r.Context(), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ListRouteMaterials: componentes da estrutura do item e a etapa em que entram.
func (h *RoutingHandler) ListRouteMaterials(w http.ResponseWriter, r *http.Request) {
	id, ok := h.routeIDDaURL(w, r)
	if !ok {
		return
	}
	result, err := h.materiaisUC.ListMaterials(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// SetRouteMaterialStep grava em que etapa um componente entra.
func (h *RoutingHandler) SetRouteMaterialStep(w http.ResponseWriter, r *http.Request) {
	id, ok := h.routeIDDaURL(w, r)
	if !ok {
		return
	}
	var dto request.SetMaterialStepDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if err := h.materiaisUC.SetMaterialStep(r.Context(), id, dto); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RoutePerformance: tempo padrão × real apontado, por etapa (?days=180).
func (h *RoutingHandler) RoutePerformance(w http.ResponseWriter, r *http.Request) {
	id, ok := h.routeIDDaURL(w, r)
	if !ok {
		return
	}
	dias, _ := strconv.Atoi(r.URL.Query().Get("days"))
	result, err := h.materiaisUC.Performance(r.Context(), id, dias)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *RoutingHandler) routeIDDaURL(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if h.materiaisUC == nil {
		jsonError(w, http.StatusServiceUnavailable, "componentes e desempenho do roteiro não configurados")
		return 0, false
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código do roteiro inválido")
		return 0, false
	}
	return id, true
}
