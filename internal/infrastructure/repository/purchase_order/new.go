package purchase_order

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// conexao é o pool ou uma transação aberta: dentro dela o Begin interno vira
// savepoint e as gravações entram no tudo-ou-nada de quem chamou.
type conexao interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PurchaseOrderRepositorySQLC struct {
	db conexao
}

// NewPurchaseOrderRepositoryTx devolve o repositório ligado a uma transação.
func NewPurchaseOrderRepositoryTx(tx pgx.Tx) *PurchaseOrderRepositorySQLC {
	return &PurchaseOrderRepositorySQLC{db: tx}
}

func NewPurchaseOrderRepositorySQLC(pool *pgxpool.Pool) *PurchaseOrderRepositorySQLC {
	return &PurchaseOrderRepositorySQLC{db: pool}
}
