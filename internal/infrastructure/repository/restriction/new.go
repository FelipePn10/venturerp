package restriction

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type RestrictionRepositorySQLC struct {
	q *sqlc.Queries
}

func NewRestrictionRepositorySQLC(q *sqlc.Queries) *RestrictionRepositorySQLC {
	return &RestrictionRepositorySQLC{q: q}
}
