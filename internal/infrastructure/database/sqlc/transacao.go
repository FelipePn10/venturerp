package sqlc

// Arquivo escrito à mão (não é gerado pelo sqlc).

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type iniciaTransacao interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// InTx roda fn numa transação: tudo ou nada. Sobre um pool abre a transação;
// sobre uma transação já aberta, um savepoint (pgx.Tx.Begin). Uma conexão que
// não sabe abrir transação recebe erro em vez de rodar sem atomicidade.
func (q *Queries) InTx(ctx context.Context, fn func(*Queries) error) error {
	b, ok := q.db.(iniciaTransacao)
	if !ok {
		return errSemTransacao
	}
	tx, err := b.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type erroSemTransacao struct{}

func (erroSemTransacao) Error() string {
	return "a conexão com o banco não permite transação para esta operação"
}

var errSemTransacao error = erroSemTransacao{}
