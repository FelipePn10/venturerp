package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	enums "github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Consulta sem resultado é "não encontrado", não falha do servidor — inclusive
// quando o repositório deixa o pgx.ErrNoRows subir sem embrulhar.
func TestRespondUseCaseErrorMapsNoRowsToNotFound(t *testing.T) {
	for nome, err := range map[string]error{
		"cru":        pgx.ErrNoRows,
		"embrulhado": fmt.Errorf("buscando máquina: %w", pgx.ErrNoRows),
		"tipado":     errorsuc.NewNotFoundError("máquina 1 não encontrada"),
	} {
		t.Run(nome, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RespondUseCaseError(rec, err)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
			}
			var corpo struct {
				Code  string `json:"code"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
				t.Fatalf("corpo inválido: %v", err)
			}
			if corpo.Code != "REGISTRO_NAO_ENCONTRADO" {
				t.Fatalf("code = %q", corpo.Code)
			}
			if corpo.Error == "" || corpo.Error == "erro interno do servidor" {
				t.Fatalf("mensagem inútil para o usuário: %q", corpo.Error)
			}
		})
	}
}

// Cadastrar um código já existente é conflito, não falha do servidor: o
// usuário precisa saber que o registro já existe.
func TestInternalErrorClassifiesDuplicateKey(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h := &BaseHandler{}
	h.InternalError(rec, req, &pgconn.PgError{Code: "23505", Message: "duplicate key"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusConflict)
	}
	if strings.Contains(rec.Body.String(), "erro interno") {
		t.Fatalf("mensagem genérica demais: %s", rec.Body.String())
	}
}

// Um erro realmente desconhecido continua sendo 500 genérico, sem vazar detalhe.
func TestInternalErrorKeepsUnknownAsInternal(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h := &BaseHandler{}
	h.InternalError(rec, req, errors.New("falha inesperada no disco"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusInternalServerError)
	}
	if strings.Contains(rec.Body.String(), "disco") {
		t.Fatalf("vazou detalhe técnico: %s", rec.Body.String())
	}
}

// Um valor fora de lista fechada devolvido pelo caso de uso precisa chegar ao
// usuário como 422 dizendo o que é aceito — antes virava 500 genérico.
func TestRespondUseCaseErrorTraduzValorForaDeListaFechada(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondUseCaseError(rec, enums.NewInvalidValue("Operador da condição (SE)", "EQ", "EQUAL", "DIFFERENT"))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, veio %d", rec.Code)
	}
	corpo := rec.Body.String()
	for _, esperado := range []string{"VALOR_NAO_ACEITO", "EQ", "EQUAL", "DIFFERENT"} {
		if !strings.Contains(corpo, esperado) {
			t.Fatalf("a resposta deveria citar %q; veio: %s", esperado, corpo)
		}
	}
}
