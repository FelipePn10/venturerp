package mrp_calculation

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type MRPCalculationRepositorySQLC struct {
	q  *sqlc.Queries
	db sqlc.DBTX
}

func NewMRPCalculationRepositorySQLC(q *sqlc.Queries, db sqlc.DBTX) *MRPCalculationRepositorySQLC {
	return &MRPCalculationRepositorySQLC{q: q, db: db}
}
