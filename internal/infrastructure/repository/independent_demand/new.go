package independent_demand

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type IndependentDemandRepositorySQLC struct {
	q *sqlc.Queries
	// db lê o acompanhamento (atendido, ordens e vendas); sem ele a demanda
	// sai sem o atendido.
	db sqlc.DBTX
}

func NewIndependentDemandRepositorySQLC(q *sqlc.Queries) *IndependentDemandRepositorySQLC {
	return &IndependentDemandRepositorySQLC{q: q}
}

// ComDB liga a leitura do acompanhamento (pool ou transação).
func (r *IndependentDemandRepositorySQLC) ComDB(db sqlc.DBTX) *IndependentDemandRepositorySQLC {
	r.db = db
	return r
}
