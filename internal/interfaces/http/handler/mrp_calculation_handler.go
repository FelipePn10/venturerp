package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *MRPCalculationHandler) Run(w http.ResponseWriter, r *http.Request) {
	var dto request.RunMRPCalculationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.runUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusAccepted, result)
}

func (h *MRPCalculationHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	itemCode, _ := strconv.ParseInt(chi.URLParam(r, "item_code"), 10, 64)
	planID, _ := strconv.ParseInt(chi.URLParam(r, "plan_code"), 10, 64)
	results, err := h.getProfileUC.Execute(r.Context(), itemCode, planID)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MRPCalculationHandler) CreateConfiguredRule(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConfiguredItemRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.configuredRulesUC.Create(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MRPCalculationHandler) ListConfiguredRules(w http.ResponseWriter, r *http.Request) {
	itemCode, _ := strconv.ParseInt(chi.URLParam(r, "item_code"), 10, 64)
	results, err := h.configuredRulesUC.ListByItem(r.Context(), itemCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MRPCalculationHandler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	planCode, err := strconv.ParseInt(chi.URLParam(r, "plan_code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid plan_code")
		return
	}
	results, err := h.listExceptionsUC.Execute(r.Context(), planCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// FirmarSugestao converts a single mrp_planned_suggestions row into a firm planned order.
// POST /api/mrp-calculation/suggestions/{code}/firm
func (h *MRPCalculationHandler) FirmarSugestao(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "invalid suggestion code")
		return
	}
	result, err := h.firmarSugestaoUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

// ListSuggestionsByPlan lists all MRP suggestions for a plan.
// GET /api/mrp-calculation/suggestions/{plan_code}
func (h *MRPCalculationHandler) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	planCode, err := strconv.ParseInt(chi.URLParam(r, "plan_code"), 10, 64)
	if err != nil || planCode <= 0 {
		security.RespondError(w, http.StatusBadRequest, "invalid plan_code")
		return
	}
	results, err := h.firmarSugestaoUC.MRPRepo.ListSuggestionsByPlan(r.Context(), planCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}
