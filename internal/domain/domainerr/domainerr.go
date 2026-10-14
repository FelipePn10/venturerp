// Package domainerr tipa os erros de regra que nascem no domínio — validação
// de entidade, registro ausente, conflito — para a borda HTTP traduzir o
// status certo (422/404/409) e mostrar a mensagem.
//
// Antes o domínio só tinha errors.New: a mensagem pronta ("código do
// almoxarifado é obrigatório") chegava à tela como "erro interno do servidor".
// O pacote fica no domínio para a regra de dependência da arquitetura valer:
// o domínio não importa a camada de aplicação; o pacote de erros da aplicação
// (usecase/errors) é que reaproveita estes tipos.
package domainerr

import (
	"database/sql"
	"errors"
)

// ValidationError: dado do pedido inválido ou regra de negócio violada (422).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// NewValidation cria um erro de validação com a mensagem para o usuário.
func NewValidation(msg string) error { return &ValidationError{Msg: msg} }

// ConflictError: o pedido conflita com o estado atual — duplicidade, registro
// em uso, chave já usada (409).
type ConflictError struct{ Msg string }

func (e *ConflictError) Error() string { return e.Msg }

// NewConflict cria um erro de conflito.
func NewConflict(msg string) error { return &ConflictError{Msg: msg} }

// NotFoundError: o registro não existe (para esta empresa) (404).
type NotFoundError struct{ Msg string }

func (e *NotFoundError) Error() string { return e.Msg }

// NewNotFound cria um erro de registro não encontrado.
func NewNotFound(msg string) error { return &NotFoundError{Msg: msg} }

// EhNaoEncontrado diz se o erro significa "o registro não existe" — o tipado
// NotFoundError ou o "nenhuma linha" do banco (pgx.ErrNoRows envolve
// sql.ErrNoRows). Serve para separar ausência legítima de falha de verdade.
func EhNaoEncontrado(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf) || errors.Is(err, sql.ErrNoRows)
}
