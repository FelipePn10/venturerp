package sales_division

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SalesDivisionRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewSalesDivisionRepositorySQLC(q *sqlc.Queries, pools ...*pgxpool.Pool) *SalesDivisionRepositorySQLC {
	repository := &SalesDivisionRepositorySQLC{q: q}
	if len(pools) > 0 {
		repository.pool = pools[0]
	}
	return repository
}
