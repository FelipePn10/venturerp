package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// filtroValido confere as datas "from"/"to" e os números informados na query
// de uma listagem. Antes, valor mal digitado era ignorado em silêncio e a
// lista vinha sem o filtro — a pessoa via "tudo" achando que tinha filtrado.
// Responde 422 e devolve false quando algo não vale.
func filtroValido(w http.ResponseWriter, r *http.Request, numeros ...string) bool {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if msg := datetime.PrimeiraInvalida(datetime.Campo("data inicial", &from), datetime.Campo("data final", &to)); msg != "" {
		respondErr(w, errorsuc.NewValidationError(msg))
		return false
	}
	for _, nome := range numeros {
		v := strings.TrimSpace(q.Get(nome))
		if v == "" {
			continue
		}
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			respondErr(w, errorsuc.NewValidationError(fmt.Sprintf("o filtro %s precisa ser um número (veio %q)", nome, v)))
			return false
		}
	}
	return true
}
