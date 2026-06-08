package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
)

type PlanningHandler struct {
	pipeline *planning_uc.RunPlanningPipelineUseCase
}

func NewPlanningHandler(pipeline *planning_uc.RunPlanningPipelineUseCase) *PlanningHandler {
	return &PlanningHandler{pipeline: pipeline}
}

// RunPipeline triggers MRP → CRP → APS for a plan and returns a consolidated
// viability assessment.
func (h *PlanningHandler) RunPipeline(w http.ResponseWriter, r *http.Request) {
	var dto request.RunPlanningPipelineDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.pipeline.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
