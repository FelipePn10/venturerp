package financial

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// conexao: o pool ou uma transação aberta — dentro de uma transação, o Begin
// interno vira savepoint e as gravações entram no tudo-ou-nada de quem chamou.
type conexao interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type FinancialRepositoryPG struct {
	pool conexao
}

var _ repository.FinancialRepository = (*FinancialRepositoryPG)(nil)

func NewFinancialRepositoryPG(pool *pgxpool.Pool) repository.FinancialRepository {
	return &FinancialRepositoryPG{pool: pool}
}

// NewFinancialRepositoryTx devolve o repositório ligado a uma transação aberta.
func NewFinancialRepositoryTx(tx pgx.Tx) repository.FinancialRepository {
	return &FinancialRepositoryPG{pool: tx}
}
