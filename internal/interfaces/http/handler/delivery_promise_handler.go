package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_uc"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type DeliveryPromiseHandler struct {
	uc *delivery_promise_uc.DeliveryPromiseUseCase
}

func NewDeliveryPromiseHandler(uc *delivery_promise_uc.DeliveryPromiseUseCase) *DeliveryPromiseHandler {
	return &DeliveryPromiseHandler{uc: uc}
}

func (h *DeliveryPromiseHandler) Occupation(w http.ResponseWriter, r *http.Request) {
	dto := request.DeliveryPromiseOccupationDTO{
		FromDate:      r.URL.Query().Get("from_date"),
		ToDate:        r.URL.Query().Get("to_date"),
		DailyCapacity: parseFloatQuery(r, "daily_capacity"),
		TankCodes:     parseInt64List(r.URL.Query()["tank_code"]),
	}
	result, err := h.uc.Occupation(r.Context(), dto)
	if err != nil {
		h.handleError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *DeliveryPromiseHandler) ReserveTank(w http.ResponseWriter, r *http.Request) {
	var dto request.DeliveryTankReservationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	result, err := h.uc.ReserveTank(r.Context(), dto)
	if err != nil {
		h.handleError(w, err)
		return
	}
	status := http.StatusOK
	if dto.Commit {
		status = http.StatusCreated
	}
	security.RespondJSON(w, status, result)
}

func (h *DeliveryPromiseHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	var dto request.DeliveryRescheduleBatchDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	result, err := h.uc.Reschedule(r.Context(), dto)
	if err != nil {
		h.handleError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *DeliveryPromiseHandler) CancelReservation(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.uc.CancelReservation(r.Context(), code); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DeliveryPromiseHandler) ExpireReservations(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if value := r.URL.Query().Get("now"); value != "" {
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "invalid now")
			return
		}
		now = parsed
	}
	count, err := h.uc.ExpireReservations(r.Context(), now)
	if err != nil {
		h.handleError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]int64{"expired": count})
}

func (h *DeliveryPromiseHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errorsuc.ErrUnauthorized):
		security.RespondError(w, http.StatusUnauthorized, err.Error())
	default:
		security.RespondError(w, http.StatusInternalServerError, err.Error())
	}
}

func parseFloatQuery(r *http.Request, name string) float64 {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func parseInt64List(values []string) []int64 {
	out := make([]int64, 0, len(values))
	for _, value := range values {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			out = append(out, parsed)
		}
	}
	return out
}
