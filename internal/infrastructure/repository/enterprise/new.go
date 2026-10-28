package enterprise

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type repositoryEnterpriseSQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewRepositoryEnterpriseSQLC(q *sqlc.Queries) *repositoryEnterpriseSQLC {
	return &repositoryEnterpriseSQLC{
		q: q,
	}
}

// ComPool habilita o cadastro completo numa transação (código livre, vínculo
// do criador e identidade fiscal).
func (r *repositoryEnterpriseSQLC) ComPool(pool *pgxpool.Pool) *repositoryEnterpriseSQLC {
	r.pool = pool
	return r
}
