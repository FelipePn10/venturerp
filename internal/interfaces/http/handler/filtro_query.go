package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

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

// filtrosDaQuery descreve os filtros de uma listagem que precisam ser número,
// lista de números ou data.
type filtrosDaQuery struct {
	Inteiros []string
	Decimais []string
	Listas   []string // inteiros separados por vírgula ou parâmetro repetido
	Datas    []string
}

// ausenteNaQuery: vazio conta como "sem filtro", e também "undefined"/"null",
// que é como a tela manda um filtro que ficou em branco.
func ausenteNaQuery(v string) bool {
	v = strings.TrimSpace(v)
	return v == "" || v == "undefined" || v == "null"
}

// queryValida confere os filtros ANTES de a listagem ler a query. Valor mal
// digitado era descartado em silêncio e a lista vinha sem o filtro — a pessoa
// via "tudo" achando que tinha filtrado. Responde 422 e devolve false.
func queryValida(w http.ResponseWriter, r *http.Request, f filtrosDaQuery) bool {
	q := r.URL.Query()
	recusa := func(msg string) bool {
		respondErr(w, errorsuc.NewValidationError(msg))
		return false
	}
	for _, nome := range f.Inteiros {
		if v := q.Get(nome); !ausenteNaQuery(v) {
			if _, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err != nil {
				return recusa(fmt.Sprintf("o filtro %s precisa ser um número inteiro (veio %q)", nome, v))
			}
		}
	}
	for _, nome := range f.Decimais {
		if v := q.Get(nome); !ausenteNaQuery(v) {
			if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
				return recusa(fmt.Sprintf("o filtro %s precisa ser um número (veio %q)", nome, v))
			}
		}
	}
	for _, nome := range f.Listas {
		for _, bruto := range q[nome] {
			for _, v := range strings.Split(bruto, ",") {
				if ausenteNaQuery(v) {
					continue
				}
				if _, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err != nil {
					return recusa(fmt.Sprintf("o filtro %s aceita só números separados por vírgula (veio %q)", nome, bruto))
				}
			}
		}
	}
	for _, nome := range f.Datas {
		if v := q.Get(nome); !ausenteNaQuery(v) {
			if msg := datetime.PrimeiraInvalida(datetime.Campo(nomeDaData(nome), &v)); msg != "" {
				return recusa(msg)
			}
		}
	}
	return true
}

// codigoDaRota lê um código numérico do caminho. Antes, texto no lugar do
// código virava 0 e a operação seguia procurando o registro 0.
func codigoDaRota(w http.ResponseWriter, r *http.Request, nome string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, nome)), 10, 64)
	if err != nil {
		respondErr(w, errorsuc.NewValidationError(fmt.Sprintf("código inválido no endereço (%q)", chi.URLParam(r, nome))))
		return 0, false
	}
	return v, true
}

// nomeDaData dá à data do filtro o nome que a pessoa vê na tela.
func nomeDaData(filtro string) string {
	switch filtro {
	case "from":
		return "data inicial"
	case "to":
		return "data final"
	}
	return "data do filtro " + filtro
}
