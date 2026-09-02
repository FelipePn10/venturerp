package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	mrprepository "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
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
		if errors.Is(err, mrprepository.ErrCalculationInProgress) {
			security.RespondErrorCode(w, http.StatusConflict, "MRP_CALCULO_EM_ANDAMENTO", "já existe um cálculo MRP em andamento para este plano")
			return
		}
		if errors.Is(err, mrp_calculation_uc.ErrInvalidPlanCode) || errors.Is(err, mrp_calculation_uc.ErrInvalidInitialOrderNumber) {
			security.RespondErrorCode(w, http.StatusUnprocessableEntity, "MRP_ENTRADA_INVALIDA", err.Error())
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusAccepted, result)
}

func (h *MRPCalculationHandler) ConsultProfile(w http.ResponseWriter, r *http.Request) {
	itemCode, itemErr := strconv.ParseInt(chi.URLParam(r, "item_code"), 10, 64)
	planCode, planErr := strconv.ParseInt(chi.URLParam(r, "plan_code"), 10, 64)
	if itemErr != nil || planErr != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "MRP_REFERENCIA_INVALIDA", "os códigos do item e do plano devem ser identificadores válidos")
		return
	}
	parseDate := func(value string) (*time.Time, error) {
		if value == "" {
			return nil, nil
		}
		date, err := time.Parse("2006-01-02", value)
		return &date, err
	}
	from, err := parseDate(r.URL.Query().Get("from"))
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "MRP_PERIODO_INVALIDO", "o campo 'from' deve estar no formato AAAA-MM-DD")
		return
	}
	to, err := parseDate(r.URL.Query().Get("to"))
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "MRP_PERIODO_INVALIDO", "o campo 'to' deve estar no formato AAAA-MM-DD")
		return
	}
	result, err := h.getProfileUC.Consult(r.Context(), itemCode, planCode, r.URL.Query().Get("position"), from, to)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MRPCalculationHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	itemCode, itemErr := strconv.ParseInt(chi.URLParam(r, "item_code"), 10, 64)
	planID, planErr := strconv.ParseInt(chi.URLParam(r, "plan_code"), 10, 64)
	if itemErr != nil || planErr != nil || itemCode <= 0 || planID <= 0 {
		security.RespondErrorCode(w, http.StatusBadRequest, "MRP_REFERENCIA_INVALIDA", "os códigos do item e do plano devem ser identificadores positivos")
		return
	}
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
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MRPCalculationHandler) ListConfiguredRules(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "item_code"), 10, 64)
	if err != nil || itemCode <= 0 {
		security.RespondErrorCode(w, http.StatusBadRequest, "MRP_ITEM_INVALIDO", "o código do item deve ser um identificador positivo")
		return
	}
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
