package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// ListPlans alimenta o modal de escolha do plano (VPRO0200). Com
// only_calculated=true devolve apenas os planos que já têm carga — os únicos
// que produzem uma exportação com conteúdo.
func (h *CRPHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	onlyCalculated := r.URL.Query().Get("only_calculated") == "true"
	result, err := h.uc.ListPlans(r.Context(), onlyCalculated)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CRPHandler) CalculateCRP(w http.ResponseWriter, r *http.Request) {
	var dto request.CalculateCRPDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.CalculateCRP(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CRPHandler) ListByPlan(w http.ResponseWriter, r *http.Request) {
	planCode, err := strconv.ParseInt(chi.URLParam(r, "planCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código de plano inválido")
		return
	}
	result, err := h.uc.ListByPlan(r.Context(), planCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CRPHandler) ListOverloadedByPlan(w http.ResponseWriter, r *http.Request) {
	planCode, err := strconv.ParseInt(chi.URLParam(r, "planCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código de plano inválido")
		return
	}
	result, err := h.uc.ListOverloadedByPlan(r.Context(), planCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
