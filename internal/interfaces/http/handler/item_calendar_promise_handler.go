package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *ItemCalendarPromiseHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.UpsertDay)

	r.Get("/{item_code}/{mask}/{year}/{month}", h.ListMonth)
	r.Get("/{item_code}/{mask}/{year}/{month}/workdays", h.GetWorkdays)
	r.Get("/{item_code}/{mask}/{year}/{month}/{day}", h.GetDay)

	r.Delete("/{item_code}/{mask}/{year}/{month}/{day}", h.DeleteDay)

	return r
}

func (h *ItemCalendarPromiseHandler) UpsertDay(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateItemCalendarDayDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "CALENDARIO_CORPO_INVALIDO", "corpo da requisição inválido")
		return
	}

	result, err := h.uc.UpsertDay(r.Context(), dto)
	if err != nil {
		h.handleError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ItemCalendarPromiseHandler) ListMonth(w http.ResponseWriter, r *http.Request) {
	itemCode, mask, year, month, ok := parseBaseParams(w, r)
	if !ok {
		return
	}

	results, err := h.uc.ListMonth(r.Context(), itemCode, mask, year, month)
	if err != nil {
		h.handleError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, results)
}

func (h *ItemCalendarPromiseHandler) GetDay(w http.ResponseWriter, r *http.Request) {
	itemCode, mask, year, month, ok := parseBaseParams(w, r)
	if !ok {
		return
	}

	day, err := strconv.Atoi(chi.URLParam(r, "day"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "dia inválido")
		return
	}

	result, err := h.uc.GetDay(r.Context(), itemCode, mask, year, month, day)
	if err != nil {
		h.handleError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ItemCalendarPromiseHandler) GetWorkdays(w http.ResponseWriter, r *http.Request) {
	itemCode, mask, year, month, ok := parseBaseParams(w, r)
	if !ok {
		return
	}

	results, err := h.uc.GetWorkdaysInMonth(r.Context(), itemCode, mask, year, month)
	if err != nil {
		h.handleError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, results)
}

func (h *ItemCalendarPromiseHandler) DeleteDay(w http.ResponseWriter, r *http.Request) {
	itemCode, mask, year, month, ok := parseBaseParams(w, r)
	if !ok {
		return
	}

	day, err := strconv.Atoi(chi.URLParam(r, "day"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "dia inválido")
		return
	}

	err = h.uc.DeleteDay(r.Context(), itemCode, mask, year, month, day)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Centraliza parsing base (reduz duplicação)
func parseBaseParams(w http.ResponseWriter, r *http.Request) (int64, string, int, int, bool) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "item_code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return 0, "", 0, 0, false
	}

	mask := chi.URLParam(r, "mask")

	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "ano inválido")
		return 0, "", 0, 0, false
	}

	month, err := strconv.Atoi(chi.URLParam(r, "month"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "mês inválido")
		return 0, "", 0, 0, false
	}

	return itemCode, mask, year, month, true
}

func (h *ItemCalendarPromiseHandler) handleError(w http.ResponseWriter, err error) {
	security.RespondUseCaseError(w, err)
}
