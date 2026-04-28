package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type CostCenterHandler struct {
	createUC *cost_center_uc.CreateCostCenterUseCase
	listUC   *cost_center_uc.ListCostCentersUseCase
	getUC    *cost_center_uc.GetCostCenterUseCase
}

func (h *CostCenterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCostCenterDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *CostCenterHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *CostCenterHandler) Get(w http.ResponseWriter, r *http.Request) {
	costCenterCode, _ := strconv.ParseInt(chi.URLParam(r, "costCenterCode"), 10, 64)
	result, err := h.getUC.Execute(r.Context(), int32(costCenterCode))
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
