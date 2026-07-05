package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_goal_uc"
	goalrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type SalesGoalHandler struct {
	uc *sales_goal_uc.UseCase
}

func NewSalesGoalHandler(uc *sales_goal_uc.UseCase) *SalesGoalHandler {
	return &SalesGoalHandler{uc: uc}
}

func (h *SalesGoalHandler) CreatePeriod(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesGoalPeriodDTO
	if decodeSalesGoal(w, r, &dto) {
		result, err := h.uc.CreatePeriod(r.Context(), dto)
		respondSalesGoal(w, http.StatusCreated, result, err)
	}
}

func (h *SalesGoalHandler) ListPeriods(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListPeriods(r.Context(), goalrepo.PeriodFilter{From: parseGoalDate(r, "from"), To: parseGoalDate(r, "to"), OnlyActive: r.URL.Query().Get("active") != "false"})
	respondSalesGoal(w, http.StatusOK, result, err)
}

func (h *SalesGoalHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesGoalDTO
	if decodeSalesGoal(w, r, &dto) {
		result, err := h.uc.CreateGoal(r.Context(), dto)
		respondSalesGoal(w, http.StatusCreated, result, err)
	}
}

func (h *SalesGoalHandler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesGoalCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateSalesGoalDTO
	if decodeSalesGoal(w, r, &dto) {
		dto.Code = code
		result, err := h.uc.UpdateGoal(r.Context(), dto)
		respondSalesGoal(w, http.StatusOK, result, err)
	}
}

func (h *SalesGoalHandler) GetGoal(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesGoalCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.GetGoal(r.Context(), code)
	respondSalesGoal(w, http.StatusOK, result, err)
}

func (h *SalesGoalHandler) ListGoals(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListGoals(r.Context(), goalrepo.GoalFilter{
		RepresentativeCode: parseGoalIntQuery(r, "representative_code"),
		PeriodCode:         parseGoalIntQuery(r, "period_code"),
		AnalysisBase:       r.URL.Query().Get("analysis_base"),
		OnlyActive:         r.URL.Query().Get("active") != "false",
	})
	respondSalesGoal(w, http.StatusOK, result, err)
}

func (h *SalesGoalHandler) AddGoalItem(w http.ResponseWriter, r *http.Request) {
	var dto request.SalesGoalItemDTO
	if decodeSalesGoal(w, r, &dto) {
		result, err := h.uc.AddGoalItem(r.Context(), dto)
		respondSalesGoal(w, http.StatusCreated, result, err)
	}
}

func (h *SalesGoalHandler) UpsertGroupTarget(w http.ResponseWriter, r *http.Request) {
	var dto request.SalesGoalGroupTargetDTO
	if decodeSalesGoal(w, r, &dto) {
		result, err := h.uc.UpsertGroupTarget(r.Context(), dto)
		respondSalesGoal(w, http.StatusCreated, result, err)
	}
}

func (h *SalesGoalHandler) AddGroupCustomer(w http.ResponseWriter, r *http.Request) {
	var dto request.SalesGoalGroupCustomerDTO
	if decodeSalesGoal(w, r, &dto) {
		result, err := h.uc.AddGroupCustomer(r.Context(), dto)
		respondSalesGoal(w, http.StatusCreated, result, err)
	}
}

func (h *SalesGoalHandler) UpsertBalance(w http.ResponseWriter, r *http.Request) {
	var dto request.SalesGoalBalanceDTO
	if decodeSalesGoal(w, r, &dto) {
		result, err := h.uc.UpsertBalance(r.Context(), dto)
		respondSalesGoal(w, http.StatusCreated, result, err)
	}
}

func (h *SalesGoalHandler) Report(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.Report(r.Context(), goalrepo.ReportFilter{
		RepresentativeCode: parseGoalIntQuery(r, "representative_code"),
		CustomerCode:       parseGoalIntQuery(r, "customer_code"),
		RegionCode:         parseGoalIntQuery(r, "region_code"),
		MicroregionCode:    parseGoalIntQuery(r, "microregion_code"),
		PeriodCode:         parseGoalIntQuery(r, "period_code"),
		From:               parseGoalDate(r, "from"),
		To:                 parseGoalDate(r, "to"),
		AnalysisBase:       r.URL.Query().Get("analysis_base"),
		Layout:             r.URL.Query().Get("layout"),
		BreakBy:            r.URL.Query().Get("break_by"),
		IncludeMissedItems: r.URL.Query().Get("include_missed_items") == "true",
	})
	respondSalesGoal(w, http.StatusOK, result, err)
}

func decodeSalesGoal(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func respondSalesGoal(w http.ResponseWriter, status int, result any, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, result)
}

func parseSalesGoalCode(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "invalid "+key)
		return 0, false
	}
	return code, true
}

func parseGoalIntQuery(r *http.Request, key string) *int64 {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseGoalDate(r *http.Request, key string) *time.Time {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &parsed
}
