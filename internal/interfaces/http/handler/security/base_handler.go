package security

import (
	"encoding/json"
	"net/http"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
)

type BaseHandler struct{}

func (h *BaseHandler) OK(w http.ResponseWriter, data any, msg ...string) {
	message := "sucesso"
	if len(msg) > 0 {
		message = msg[0]
	}
	WriteSuccess(w, http.StatusOK, data, message)
}

func (h *BaseHandler) Created(w http.ResponseWriter, data any, msg ...string) {
	message := "criado com sucesso"
	if len(msg) > 0 {
		message = msg[0]
	}
	WriteSuccess(w, http.StatusCreated, data, message)
}

func (h *BaseHandler) BadRequest(w http.ResponseWriter, message string, details ...any) {
	WriteError(w, http.StatusBadRequest, "bad_request", message, details...)
}

func (h *BaseHandler) NotFound(w http.ResponseWriter, message ...string) {
	msg := "recurso não encontrado"
	if len(message) > 0 {
		msg = message[0]
	}
	WriteError(w, http.StatusNotFound, "not_found", msg)
}

// InternalError logs the real error (with request_id from context) and returns
// a generic message to the client — never leaking internal details.
// InternalError é o destino dos erros que o handler não classificou. Antes de
// assumir falha do servidor, ele deixa RespondUseCaseError reconhecer o que
// for conhecido — chave duplicada vira 409 "já existe", registro ausente vira
// 404, violação de domínio vira 422. Sem isso, cadastrar um código repetido
// respondia "ocorreu um erro interno", que não diz ao usuário o que houve.
func (h *BaseHandler) InternalError(w http.ResponseWriter, r *http.Request, err error) {
	applogger.FromContext(r.Context()).Error(
		"internal server error",
		"error", err,
	)
	if isClassified(err) {
		RespondUseCaseError(w, err)
		return
	}
	WriteError(w, http.StatusInternalServerError, "internal_error", "ocorreu um erro interno")
}

// Conflict returns 409 for requests that clash with existing state, most
// commonly a duplicate unique key.
func (h *BaseHandler) Conflict(w http.ResponseWriter, message ...string) {
	msg := "recurso já existe"
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
	if status >= http.StatusInternalServerError {
		message = "erro interno do servidor"
	}
	RespondErrorCode(w, status, defaultErrorCode(status), message)
}

func RespondErrorCode(w http.ResponseWriter, status int, code, message string) {
	RespondJSON(w, status, map[string]string{"error": message, "code": code})
}

func defaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "REQUISICAO_INVALIDA"
	case http.StatusForbidden:
		return "ACESSO_NEGADO"
	case http.StatusNotFound:
		return "REGISTRO_NAO_ENCONTRADO"
	case http.StatusConflict:
		return "CONFLITO_DE_DOMINIO"
	case http.StatusUnprocessableEntity:
		return "VALIDACAO_DE_DOMINIO"
	default:
		return "ERRO_INTERNO"
	}
}
