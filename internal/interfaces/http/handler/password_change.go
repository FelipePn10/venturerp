package handler

import (
	"encoding/json"
	"errors"
	"io"
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

// Request abre a solicitação. O corpo é opcional: {"reason": "..."}.
func (h *PasswordChangeHandler) Request(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			h.BadRequest(w, "corpo da requisição inválido")
			return
		}
	}
	request, err := h.uc.Request(r.Context(), body.Reason)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.Created(w, request, "solicitação de troca de senha registrada")
}

func (h *PasswordChangeHandler) List(w http.ResponseWriter, r *http.Request) {
	requests, err := h.uc.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.OK(w, requests)
}

// Mine: as solicitações do usuário logado (qualquer perfil).
func (h *PasswordChangeHandler) Mine(w http.ResponseWriter, r *http.Request) {
	requests, err := h.uc.Mine(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.OK(w, requests)
}

// errIdentificadorInvalido: o id da solicitação é um UUID. Antes um id fora do
// formato ("0", "10") caía no erro genérico e respondia 500.
var errIdentificadorInvalido = errors.New("identificador da solicitação de troca de senha inválido")

func passwordRequestID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "requestID"))
	if err != nil {
		return uuid.Nil, errIdentificadorInvalido
	}
	return id, nil
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
		h.BadRequest(w, "corpo da requisição inválido")
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
		h.BadRequest(w, "corpo da requisição inválido")
		return
	}
	if body.NewPassword != body.ConfirmPassword {
		h.BadRequest(w, "a confirmação não confere com a nova senha")
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
	h.OK(w, map[string]string{"status": "PASSWORD_CHANGED"}, "senha alterada; entre novamente")
}

func (h *PasswordChangeHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, user_uc.ErrPasswordChangeForbidden), errors.Is(err, user_uc.ErrPasswordChangeOwn):
		security.WriteError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, user_uc.ErrCurrentPasswordInvalid):
		// 422, e não 401: a sessão continua válida; 401 faz o app encerrar a
		// sessão de quem só digitou a senha atual errada.
		security.WriteError(w, http.StatusUnprocessableEntity, "invalid_current_password", err.Error())
	case errors.Is(err, user_uc.ErrPasswordChangeNotFound):
		h.NotFound(w, err.Error())
	case errors.Is(err, user_uc.ErrPasswordChangeActive), errors.Is(err, user_uc.ErrPasswordChangeDecided),
		errors.Is(err, user_uc.ErrPasswordChangeExpired), errors.Is(err, user_uc.ErrPasswordChangeNotReady):
		h.Conflict(w, err.Error())
	case errors.Is(err, user_uc.ErrWeakPassword), errors.Is(err, user_uc.ErrSamePassword),
		errors.Is(err, user_uc.ErrPasswordChangeInvalid), errors.Is(err, user_uc.ErrRejectionReason),
		errors.Is(err, user_uc.ErrRequestReason), errors.Is(err, errIdentificadorInvalido):
		h.BadRequest(w, err.Error())
	default:
		h.InternalError(w, r, err)
	}
}
