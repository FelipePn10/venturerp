package item

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type RepositoryItemSQLC struct {
	q *sqlc.Queries
	// pool, quando presente, faz a alteração do item numa transação que leva o
	// autor ao histórico do cadastro (gatilho trg_items_historico).
	pool *pgxpool.Pool
}

// ComPool liga a transação com autor na alteração do item.
func (r *RepositoryItemSQLC) ComPool(pool *pgxpool.Pool) *RepositoryItemSQLC {
	r.pool = pool
	return r
}

func NewRepositoryItemSQLC(q *sqlc.Queries) *RepositoryItemSQLC {
	return &RepositoryItemSQLC{
		q: q,
	}
}
