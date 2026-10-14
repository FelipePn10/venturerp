package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *ProductionPlanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateProductionPlanDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		if errors.Is(err, entity.ErrInvalidPlan) {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			respondErrStatus(w, http.StatusConflict, err)
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionPlanHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *ProductionPlanHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		respondErrStatus(w, http.StatusNotFound, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionPlanHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateProductionPlanDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		if errors.Is(err, entity.ErrInvalidPlan) {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionPlanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	if err := h.deleteUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "plan deleted"})
}

func (h *ProductionPlanHandler) ListInterFactories(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	result, err := h.interFactoryUC.List(r.Context(), code)
	if err != nil {
		respondErrStatus(w, http.StatusNotFound, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionPlanHandler) ReplaceInterFactories(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto request.ReplaceProductionPlanInterFactoriesDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.interFactoryUC.Replace(r.Context(), code, dto)
	if err != nil {
		if errors.Is(err, entity.ErrInvalidPlan) {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondErr(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
