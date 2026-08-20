package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/notification_uc"
	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type NotificationHandler struct{ service *notification_uc.Service }

func NewNotificationHandler(service *notification_uc.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func notificationError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "não foi possível processar a solicitação"
	if errors.Is(err, notification_uc.ErrUnauthorized) {
		status = http.StatusForbidden
		message = "operação não autorizada"
	}
	if errors.Is(err, notificationentity.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		status, message = http.StatusNotFound, err.Error()
	} else if errors.Is(err, notificationentity.ErrConflict) {
		status, message = http.StatusConflict, err.Error()
	} else if errors.Is(err, notificationentity.ErrValidation) {
		status, message = http.StatusUnprocessableEntity, err.Error()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "registro já existe"
		case "23503", "23514", "23502", "22P02":
			status, message = http.StatusUnprocessableEntity, "referência ou valor inválido"
		}
	}
	jsonError(w, status, message)
}
func decodeNotification(w http.ResponseWriter, r *http.Request, dst any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		jsonError(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}
func page(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return limit, offset
}

func (h *NotificationHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListEvents(r.Context())
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) ListEligibleUsers(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListEligibleUsers(r.Context())
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) ListEligibleDepartments(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListEligibleDepartments(r.Context())
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetSettings(r.Context())
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var in notificationentity.Settings
	if !decodeNotification(w, r, &in) {
		return
	}
	if err := h.service.SaveSettings(r.Context(), in); err != nil {
		notificationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *NotificationHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListSubscriptions(r.Context())
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var in notificationentity.Subscription
	if !decodeNotification(w, r, &in) {
		return
	}
	in.ID = uuid.Nil
	id, err := h.service.SaveSubscription(r.Context(), in)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]any{"id": id})
}
func (h *NotificationHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in notificationentity.Subscription
	if !decodeNotification(w, r, &in) {
		return
	}
	in.ID = id
	if _, err = h.service.SaveSubscription(r.Context(), in); err != nil {
		notificationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *NotificationHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err = h.service.DeleteSubscription(r.Context(), id); err != nil {
		notificationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *NotificationHandler) TestEmail(w http.ResponseWriter, r *http.Request) {
	if err := h.service.TestEmail(r.Context()); err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, map[string]string{"status": "PENDENTE"})
}
func (h *NotificationHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	l, o := page(r)
	v, err := h.service.ListRecords(r.Context(), "deliveries", l, o)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err = h.service.Retry(r.Context(), id); err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, map[string]string{"status": "PENDENTE"})
}
func (h *NotificationHandler) ListDeadLetters(w http.ResponseWriter, r *http.Request) {
	l, o := page(r)
	v, err := h.service.ListRecords(r.Context(), "dead_letters", l, o)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	l, o := page(r)
	v, err := h.service.ListRecords(r.Context(), "alerts", l, o)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}
func (h *NotificationHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "id inválido")
		return
	}
	v, err := h.service.GetAlert(r.Context(), id)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, v)
}

func (h *NotificationHandler) CreateCycleCount(w http.ResponseWriter, r *http.Request) {
	var in notificationentity.CycleCount
	if !decodeNotification(w, r, &in) {
		return
	}
	created, err := h.service.CreateCycleCount(r.Context(), in)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, created)
}
func (h *NotificationHandler) ListCycleCounts(w http.ResponseWriter, r *http.Request) {
	l, o := page(r)
	items, err := h.service.ListCycleCounts(r.Context(), l, o)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, items)
}
func (h *NotificationHandler) GetCycleCount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "id inválido")
		return
	}
	item, err := h.service.GetCycleCount(r.Context(), id)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, item)
}
func (h *NotificationHandler) TransitionCycleCount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in struct {
		State           notificationentity.CycleCountState `json:"state"`
		CountedQuantity *string                            `json:"counted_quantity"`
		Reason          string                             `json:"reason"`
	}
	if !decodeNotification(w, r, &in) {
		return
	}
	item, err := h.service.TransitionCycleCount(r.Context(), id, in.State, in.CountedQuantity, in.Reason)
	if err != nil {
		notificationError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, item)
}
