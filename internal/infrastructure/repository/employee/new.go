package employee

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type RepositoryEmployeeSQLC struct {
	q *sqlc.Queries
}

func NewRepositoryEmployeeSQLC(q *sqlc.Queries) *RepositoryEmployeeSQLC {
	return &RepositoryEmployeeSQLC{q: q}
}
