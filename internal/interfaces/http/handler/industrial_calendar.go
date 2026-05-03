package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *IndustrialCalendarHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.CreateDay)
	r.Get("/{year}/{month}", h.GetMonth)
	r.Get("/workdays/{year}/{month}", h.GetWorkdays)
	return r
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
	year, _ := strconv.Atoi(chi.URLParam(r, "year"))
	month, _ := strconv.Atoi(chi.URLParam(r, "month"))
	results, err := h.uc.GetMonth(r.Context(), year, month)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *IndustrialCalendarHandler) GetWorkdays(w http.ResponseWriter, r *http.Request) {
	year, _ := strconv.Atoi(chi.URLParam(r, "year"))
	month, _ := strconv.Atoi(chi.URLParam(r, "month"))
	results, err := h.uc.GetWorkdaysInMonth(r.Context(), year, month)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}
