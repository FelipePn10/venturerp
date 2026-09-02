package security

import (
	"errors"
	"log/slog"
	"net/http"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

// RespondUseCaseError maps an application/use-case error to the appropriate
// HTTP status code and writes a JSON error response. It recognises the typed
// errors from the errorsuc package as well as raw Postgres errors that leak
// through repository wrapping (unique violation -> 409, not-null/check/fk ->
// 422), so callers no longer collapse every failure to 500.
func RespondUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errorsuc.ErrUnauthorized):
		RespondErrorCode(w, http.StatusForbidden, "ACESSO_NEGADO", "usuário não autorizado para esta operação")
		return
	}

	if v, ok := errorsuc.AsValidation(err); ok {
		RespondErrorCode(w, http.StatusUnprocessableEntity, "VALIDACAO_DE_DOMINIO", v.Error())
		return
	}
	if c, ok := errorsuc.AsConflict(err); ok {
		RespondErrorCode(w, http.StatusConflict, "CONFLITO_DE_DOMINIO", c.Error())
		return
	}
	if n, ok := errorsuc.AsNotFound(err); ok {
		RespondErrorCode(w, http.StatusNotFound, "REGISTRO_NAO_ENCONTRADO", n.Error())
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			RespondError(w, http.StatusConflict, "já existe um registro com os dados informados")
			return
		case "23503": // foreign_key_violation
			RespondError(w, http.StatusUnprocessableEntity, "um dos vínculos informados não existe na empresa autenticada")
			return
		case "23502", "23514": // not_null_violation, check_violation
			RespondError(w, http.StatusUnprocessableEntity, "há um campo obrigatório ausente ou inválido")
			return
		}
	}

	slog.Error("erro interno em caso de uso", "error", err)
	RespondError(w, http.StatusInternalServerError, err.Error())
}
