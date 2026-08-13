package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *IndustrialCalendarHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/create", h.CreateDay)
	r.Post("/generate", h.Generate)
	r.Post("/generate/{year}/{month}", h.GenerateMonth)
	r.Get("/month/{year}/{month}", h.GetMonth)
	r.Get("/workdays/{year}/{month}", h.GetWorkdays)
	return r
}

func (h *IndustrialCalendarHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateIndustrialCalendarDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	result, err := h.uc.Generate(r.Context(), dto)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, industrial_calendar_uc.ErrInvalidCalendarDate) {
			status = http.StatusUnprocessableEntity
		}
		security.RespondError(w, status, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func calendarYearMonth(r *http.Request) (int, int, error) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		return 0, 0, err
	}
	month, err := strconv.Atoi(chi.URLParam(r, "month"))
	if err != nil {
		return 0, 0, err
	}
	return year, month, nil
}

func (h *IndustrialCalendarHandler) GenerateMonth(w http.ResponseWriter, r *http.Request) {
	year, month, err := calendarYearMonth(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "ano ou mes invalido")
		return
	}
	result, err := h.uc.GenerateMonth(r.Context(), year, month)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, industrial_calendar_uc.ErrInvalidCalendarDate) {
			status = http.StatusUnprocessableEntity
		}
		security.RespondError(w, status, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndustrialCalendarHandler) CreateDay(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCalendarDayDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateDay(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *IndustrialCalendarHandler) GetMonth(w http.ResponseWriter, r *http.Request) {
	year, month, parseErr := calendarYearMonth(r)
	if parseErr != nil {
		security.RespondError(w, http.StatusBadRequest, "ano ou mes invalido")
		return
	}
	results, err := h.uc.GetMonth(r.Context(), year, month)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *IndustrialCalendarHandler) GetWorkdays(w http.ResponseWriter, r *http.Request) {
	year, month, parseErr := calendarYearMonth(r)
	if parseErr != nil {
		security.RespondError(w, http.StatusBadRequest, "ano ou mes invalido")
		return
	}
	results, err := h.uc.GetWorkdaysInMonth(r.Context(), year, month)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}
