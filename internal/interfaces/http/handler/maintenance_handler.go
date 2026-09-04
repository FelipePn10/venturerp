package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/maintenance_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type MaintenanceHandler struct {
	*security.BaseHandler
	uc *maintenance_uc.MaintenanceUseCase
}

func NewMaintenanceHandler(uc *maintenance_uc.MaintenanceUseCase) *MaintenanceHandler {
	return &MaintenanceHandler{BaseHandler: &security.BaseHandler{}, uc: uc}
}

func (h *MaintenanceHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var dto maintenance_uc.CreatePlanDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreatePlan(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MaintenanceHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "PLANO_ID_INVALIDO", "identificador do plano inválido")
		return
	}
	result, err := h.uc.GetPlan(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MaintenanceHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("active") != "false"
	results, err := h.uc.ListPlans(r.Context(), onlyActive)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, paginate(w, r, results))
}

func (h *MaintenanceHandler) ListPlansByMachine(w http.ResponseWriter, r *http.Request) {
	machineID, err := strconv.ParseInt(chi.URLParam(r, "machineId"), 10, 64)
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "MAQUINA_ID_INVALIDO", "identificador da máquina inválido")
		return
	}
	results, err := h.uc.ListPlansByMachine(r.Context(), machineID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, paginate(w, r, results))
}

func (h *MaintenanceHandler) DeactivatePlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "PLANO_ID_INVALIDO", "identificador do plano inválido")
		return
	}
	if err := h.uc.DeactivatePlan(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

func (h *MaintenanceHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var dto maintenance_uc.CreateOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateOrder(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MaintenanceHandler) AdvanceOrder(w http.ResponseWriter, r *http.Request) {
	var dto maintenance_uc.AdvanceOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.AdvanceOrder(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MaintenanceHandler) ListOrdersByPlan(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.ParseInt(chi.URLParam(r, "planId"), 10, 64)
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "PLANO_ID_INVALIDO", "identificador do plano inválido")
		return
	}
	results, err := h.uc.ListOrdersByPlan(r.Context(), planID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, paginate(w, r, results))
}

func (h *MaintenanceHandler) ListOrdersByWorkCenter(w http.ResponseWriter, r *http.Request) {
	wcID, err := strconv.ParseInt(chi.URLParam(r, "wcId"), 10, 64)
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "CENTRO_TRABALHO_ID_INVALIDO", "identificador do centro de trabalho inválido")
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	from, fromErr := time.Parse("2006-01-02", fromStr)
	if fromErr != nil {
		security.RespondErrorCode(w, http.StatusUnprocessableEntity, "PERIODO_INVALIDO", "o campo 'from' deve estar no formato AAAA-MM-DD")
		return
	}
	to, toErr := time.Parse("2006-01-02", toStr)
	if toStr != "" && toErr != nil {
		security.RespondErrorCode(w, http.StatusUnprocessableEntity, "PERIODO_INVALIDO", "o campo 'to' deve estar no formato AAAA-MM-DD")
		return
	}
	if to.IsZero() {
		to = from.AddDate(0, 1, 0)
	}
	results, err := h.uc.ListOrdersByWorkCenter(r.Context(), wcID, from, to)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, paginate(w, r, results))
}

func (h *MaintenanceHandler) GenerateOrders(w http.ResponseWriter, r *http.Request) {
	horizonDays := 30
	if v := r.URL.Query().Get("horizon_days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			horizonDays = n
		}
	}
	count, err := h.uc.GenerateOrders(r.Context(), horizonDays)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]int{"orders_created": count})
}
