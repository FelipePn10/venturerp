package cost_center

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type CostCenterRepositorySQLC struct {
	q *sqlc.Queries
}

func NewCostCenterRepositorySQLC(q *sqlc.Queries) *CostCenterRepositorySQLC {
	return &CostCenterRepositorySQLC{q: q}
}
