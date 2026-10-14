package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type CostCenterHandler struct {
	createUC *cost_center_uc.CreateCostCenterUseCase
	listUC   *cost_center_uc.ListCostCentersUseCase
	getUC    *cost_center_uc.GetCostCenterUseCase
}

func (h *CostCenterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCostCenterDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *CostCenterHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *CostCenterHandler) Get(w http.ResponseWriter, r *http.Request) {
	costCenterCode, ok := codigoDaRota(w, r, "costCenterCode")
	if !ok {
		return
	}
	result, err := h.getUC.Execute(r.Context(), int32(costCenterCode))
	if err != nil {
		respondErrStatus(w, http.StatusNotFound, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
