package security

import (
	"errors"
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
		RespondError(w, http.StatusForbidden, err.Error())
		return
	}

	if v, ok := errorsuc.AsValidation(err); ok {
		RespondError(w, http.StatusUnprocessableEntity, v.Error())
		return
	}
	if c, ok := errorsuc.AsConflict(err); ok {
		RespondError(w, http.StatusConflict, c.Error())
		return
	}
	if n, ok := errorsuc.AsNotFound(err); ok {
		RespondError(w, http.StatusNotFound, n.Error())
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			RespondError(w, http.StatusConflict, "resource already exists")
			return
		case "23503": // foreign_key_violation
			RespondError(w, http.StatusUnprocessableEntity, "referenced resource not found")
			return
		case "23502", "23514": // not_null_violation, check_violation
			RespondError(w, http.StatusUnprocessableEntity, "invalid or missing required field")
			return
		}
	}

	RespondError(w, http.StatusInternalServerError, err.Error())
}
