package production_plan

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type ProductionPlanRepositorySQLC struct {
	q *sqlc.Queries
}

func NewProductionPlanRepositorySQLC(q *sqlc.Queries) *ProductionPlanRepositorySQLC {
	return &ProductionPlanRepositorySQLC{q: q}
}
