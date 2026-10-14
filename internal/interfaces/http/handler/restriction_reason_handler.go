package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *RestrictionReasonHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRestrictionReasonDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto.Description, dto.Situation)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *RestrictionReasonHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *RestrictionReasonHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
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

func (h *RestrictionReasonHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto request.UpdateRestrictionReasonDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	dto.Code = code
	result, err := h.updateUC.Execute(r.Context(), &entity.RestrictionReason{
		Code:        dto.Code,
		Description: dto.Description,
		Situation:   dto.Situation,
	})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RestrictionReasonHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	if err := h.deleteUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
