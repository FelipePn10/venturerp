package mrp_calculation

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type MRPCalculationRepositorySQLC struct {
	q *sqlc.Queries
}

func NewMRPCalculationRepositorySQLC(q *sqlc.Queries) *MRPCalculationRepositorySQLC {
	return &MRPCalculationRepositorySQLC{q: q}
}
