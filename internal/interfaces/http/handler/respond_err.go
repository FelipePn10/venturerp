package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// respondErr responde o erro de um caso de uso sem expor o banco.
//
// Erro tipado (validação, não encontrado, conflito, sem permissão), erro do
// banco e "nenhuma linha" seguem a tradução padrão: status certo e mensagem
// legível — FK, enum, unique e check viram frases, não "ERROR: … (SQLSTATE
// …)". O resto continua 422 com a mensagem, como os handlers sempre
// responderam: muitos casos de uso ainda devolvem a validação como erro
// simples, e ela não pode virar "erro interno".
func respondErr(w http.ResponseWriter, err error) {
	respondErrStatus(w, http.StatusUnprocessableEntity, err)
}

// respondErrStatus é respondErr mantendo o status que o handler já usava (404,
// 409…) para o erro simples.
func respondErrStatus(w http.ResponseWriter, status int, err error) {
	var pgErr *pgconn.PgError
	_, validacao := errorsuc.AsValidation(err)
	_, conflito := errorsuc.AsConflict(err)
	_, ausente := errorsuc.AsNotFound(err)
	if errors.As(err, &pgErr) || errors.Is(err, pgx.ErrNoRows) || validacao || conflito || ausente || errors.Is(err, errorsuc.ErrUnauthorized) {
		security.RespondUseCaseError(w, err)
		return
	}
	respondErr(w, err)
}
