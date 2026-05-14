package planning_params

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type PlanningParamRepositorySQLC struct {
	q *sqlc.Queries
}

func NewPlanningParamRepositorySQLC(q *sqlc.Queries) *PlanningParamRepositorySQLC {
	return &PlanningParamRepositorySQLC{q: q}
}
