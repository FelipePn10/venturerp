package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *IndependentDemandHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateIndependentDemandDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *IndependentDemandHandler) Update(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")

	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	var dto request.UpdateIndependentDemandDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	dto.CodeDemand = code

	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *IndependentDemandHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	itemStr := chi.URLParam(r, "itemCode")

	itemCode, err := strconv.ParseInt(itemStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código de item inválido")
		return
	}

	result, err := h.listByItemUC.Execute(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")

	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	result, err := h.getByCodeUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) ListFromDate(w http.ResponseWriter, r *http.Request) {
	dateStr := chi.URLParam(r, "date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "formato de data inválido")
		return
	}

	result, err := h.listFromDateUC.Execute(r.Context(), date)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) Delete(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")

	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	err = h.deleteUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
