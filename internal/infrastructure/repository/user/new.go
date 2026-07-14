package user

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repositoryUserSQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewRepositoryUserSQLC(q *sqlc.Queries, pool *pgxpool.Pool) *repositoryUserSQLC {
	return &repositoryUserSQLC{
		q: q, pool: pool,
	}
}
