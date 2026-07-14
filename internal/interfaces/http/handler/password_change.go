package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PasswordChangeHandler struct {
	*security.BaseHandler
	uc *user_uc.PasswordChangeUseCase
}

func NewPasswordChangeHandler(uc *user_uc.PasswordChangeUseCase) *PasswordChangeHandler {
	return &PasswordChangeHandler{BaseHandler: &security.BaseHandler{}, uc: uc}
}

func (h *PasswordChangeHandler) Request(w http.ResponseWriter, r *http.Request) {
	request, err := h.uc.Request(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.Created(w, request, "password change requested")
}

func (h *PasswordChangeHandler) List(w http.ResponseWriter, r *http.Request) {
	requests, err := h.uc.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.OK(w, requests)
}

func passwordRequestID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "requestID"))
}

func (h *PasswordChangeHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := passwordRequestID(r)
	if err == nil {
		err = h.uc.Approve(r.Context(), id)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.OK(w, map[string]string{"status": "APPROVED"})
}

func (h *PasswordChangeHandler) Reject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}
	id, err := passwordRequestID(r)
	if err == nil {
		err = h.uc.Reject(r.Context(), id, body.Reason)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.OK(w, map[string]string{"status": "REJECTED"})
}

func (h *PasswordChangeHandler) Complete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}
	if body.NewPassword != body.ConfirmPassword {
		h.BadRequest(w, "password confirmation does not match")
		return
	}
	id, err := passwordRequestID(r)
	if err == nil {
		err = h.uc.Complete(r.Context(), id, body.CurrentPassword, body.NewPassword)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.OK(w, map[string]string{"status": "PASSWORD_CHANGED"}, "password changed; authenticate again")
}

func (h *PasswordChangeHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, user_uc.ErrPasswordChangeForbidden):
		security.WriteError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, user_uc.ErrCurrentPasswordInvalid):
		security.WriteError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, user_uc.ErrWeakPassword), errors.Is(err, user_uc.ErrPasswordChangeInvalid):
		h.BadRequest(w, err.Error())
	default:
		h.InternalError(w, r, err)
	}
}
