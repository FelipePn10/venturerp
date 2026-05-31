package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/location_uc"
	"github.com/go-chi/chi/v5"
)

type LocationHandler struct {
	uc *location_uc.LocationUseCase
}

func NewLocationHandler(uc *location_uc.LocationUseCase) *LocationHandler {
	return &LocationHandler{uc: uc}
}

// ─── Countries ────────────────────────────────────────────────────────────────

func (h *LocationHandler) CreateCountry(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCountryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateCountry(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *LocationHandler) UpdateCountry(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateCountryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateCountry(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *LocationHandler) GetCountry(w http.ResponseWriter, r *http.Request) {
	sigla := chi.URLParam(r, "sigla")
	result, err := h.uc.GetCountryBySigla(r.Context(), sigla)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *LocationHandler) ListCountries(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListCountries(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── UFs ──────────────────────────────────────────────────────────────────────

func (h *LocationHandler) CreateUF(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateUFDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateUF(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *LocationHandler) UpdateUF(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateUFDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateUF(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *LocationHandler) GetUF(w http.ResponseWriter, r *http.Request) {
	sigla := chi.URLParam(r, "sigla")
	result, err := h.uc.GetUFBySigla(r.Context(), sigla)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *LocationHandler) ListUFs(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListUFs(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *LocationHandler) ListUFsByCountry(w http.ResponseWriter, r *http.Request) {
	sigla := chi.URLParam(r, "sigla")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListUFsByCountry(r.Context(), sigla, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
