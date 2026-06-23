package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

// ─── plans ────────────────────────────────────────────────────────────────────

func (h *CuttingPlanHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCuttingPlanDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CuttingPlanHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	onlyOpen := r.URL.Query().Get("only_open") == "true"
	result, err := h.uc.List(r.Context(), onlyOpen)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CuttingPlanHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetDetail(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CuttingPlanHandler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.Delete(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Optimize runs the nesting optimiser and returns the computed cutting plan.
func (h *CuttingPlanHandler) Optimize(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.Optimize(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// Release firms the plan: posts the real stock baixa, generates remnants and
// records the consumption trail.
func (h *CuttingPlanHandler) Release(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.ReleasePlan(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// GenerateFromOrders builds cutting plans automatically from production/planned
// orders, aggregating parts of the same raw material into one plan per material.
func (h *CuttingPlanHandler) GenerateFromOrders(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateCuttingFromOrdersDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.demand.GenerateFromOrders(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ExportMap renders the plan's cutting map (?format=svg|dxf|pdf).
func (h *CuttingPlanHandler) ExportMap(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "svg"
	}
	data, contentType, err := h.uc.ExportMap(r.Context(), id, format)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\"plano-corte-"+strconv.FormatInt(id, 10)+"."+format+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GetProgram returns the ordered cut program for the plan.
func (h *CuttingPlanHandler) GetProgram(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetProgram(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// Schedule books the plan onto its machine's calendar.
func (h *CuttingPlanHandler) Schedule(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.ScheduleOnMachine(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ListOrderCosts returns the per-order cost allocation of a firmed plan.
func (h *CuttingPlanHandler) ListOrderCosts(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.ListOrderCosts(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── settings ─────────────────────────────────────────────────────────────────

func (h *CuttingPlanHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.GetSettings(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CuttingPlanHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var dto request.CuttingSettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateSettings(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── remnants ─────────────────────────────────────────────────────────────────

func (h *CuttingPlanHandler) ListRemnants(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(r.URL.Query().Get("item_code"), 10, 64)
	if err != nil || itemCode <= 0 {
		jsonError(w, http.StatusBadRequest, "item_code must be a positive integer")
		return
	}
	onlyAvailable := r.URL.Query().Get("only_available") == "true"
	result, err := h.uc.ListRemnants(r.Context(), itemCode, onlyAvailable)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── parts ────────────────────────────────────────────────────────────────────

func (h *CuttingPlanHandler) AddPart(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.AddCuttingPlanPartDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.PlanID = id
	result, err := h.uc.AddPart(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CuttingPlanHandler) RemovePart(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "partId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid partId")
		return
	}
	if err := h.uc.RemovePart(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── stock pieces ─────────────────────────────────────────────────────────────

func (h *CuttingPlanHandler) AddStock(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.AddCuttingStockPieceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.PlanID = id
	result, err := h.uc.AddStock(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CuttingPlanHandler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "stockId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid stockId")
		return
	}
	if err := h.uc.RemoveStock(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func planID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}
