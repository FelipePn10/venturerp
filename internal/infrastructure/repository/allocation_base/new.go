package allocation_base

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type AllocationBaseRepositorySQLC struct {
	q *sqlc.Queries
}

func NewAllocationBaseRepositorySQLC(q *sqlc.Queries) *AllocationBaseRepositorySQLC {
	return &AllocationBaseRepositorySQLC{q: q}
}
