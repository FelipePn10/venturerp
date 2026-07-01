package security

import (
	"encoding/json"
	"net/http"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
)

type BaseHandler struct{}

func (h *BaseHandler) OK(w http.ResponseWriter, data any, msg ...string) {
	message := "success"
	if len(msg) > 0 {
		message = msg[0]
	}
	WriteSuccess(w, http.StatusOK, data, message)
}

func (h *BaseHandler) Created(w http.ResponseWriter, data any, msg ...string) {
	message := "created"
	if len(msg) > 0 {
		message = msg[0]
	}
	WriteSuccess(w, http.StatusCreated, data, message)
}

func (h *BaseHandler) BadRequest(w http.ResponseWriter, message string, details ...any) {
	WriteError(w, http.StatusBadRequest, "bad_request", message, details...)
}

func (h *BaseHandler) NotFound(w http.ResponseWriter, message ...string) {
	msg := "resource not found"
	if len(message) > 0 {
		msg = message[0]
	}
	WriteError(w, http.StatusNotFound, "not_found", msg)
}

// InternalError logs the real error (with request_id from context) and returns
// a generic message to the client — never leaking internal details.
func (h *BaseHandler) InternalError(w http.ResponseWriter, r *http.Request, err error) {
	applogger.FromContext(r.Context()).Error(
		"internal server error",
		"error", err,
	)
	WriteError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
}

// Conflict returns 409 for requests that clash with existing state, most
// commonly a duplicate unique key.
func (h *BaseHandler) Conflict(w http.ResponseWriter, message ...string) {
	msg := "resource already exists"
	if len(message) > 0 {
		msg = message[0]
	}
	WriteError(w, http.StatusConflict, "conflict", msg)
}

func (h *BaseHandler) UnprocessableEntity(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": http.StatusUnprocessableEntity,
	})
}

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"error": message})
}
