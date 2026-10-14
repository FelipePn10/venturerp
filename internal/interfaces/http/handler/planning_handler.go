package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type PlanningHandler struct {
	pipeline *planning_uc.RunPlanningPipelineUseCase
	autoRun  *planning_uc.AutoRunPlanningUseCase
}

func NewPlanningHandler(pipeline *planning_uc.RunPlanningPipelineUseCase, autoRun *planning_uc.AutoRunPlanningUseCase) *PlanningHandler {
	return &PlanningHandler{pipeline: pipeline, autoRun: autoRun}
}

// RunPipeline triggers MRP → CRP → APS for a plan and returns a consolidated
// viability assessment.
func (h *PlanningHandler) RunPipeline(w http.ResponseWriter, r *http.Request) {
	var dto request.RunPlanningPipelineDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.pipeline.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// GetAutoRun devolve a parametrização da execução automática da empresa.
func (h *PlanningHandler) GetAutoRun(w http.ResponseWriter, r *http.Request) {
	out, err := h.autoRun.GetSettings(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// UpdateAutoRun grava a janela noturna; quem salva vira o responsável.
func (h *PlanningHandler) UpdateAutoRun(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateAutoRunSettingsDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.autoRun.UpdateSettings(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// RunAutoRunNow executa agora com a parametrização salva.
func (h *PlanningHandler) RunAutoRunNow(w http.ResponseWriter, r *http.Request) {
	out, err := h.autoRun.RunNow(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// ListRuns devolve o histórico de execuções do planejamento (?limit=, padrão 50).
func (h *PlanningHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			security.RespondErrorCode(w, http.StatusBadRequest, "PLANEJAMENTO_LIMITE_INVALIDO", "limit deve ser um número inteiro")
			return
		}
		limit = n
	}
	out, err := h.autoRun.ListRuns(r.Context(), limit)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}
