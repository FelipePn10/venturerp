package sales_forecast

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
import "github.com/jackc/pgx/v5/pgxpool"

type SalesForecastRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewSalesForecastRepositorySQLC(q *sqlc.Queries, pool ...*pgxpool.Pool) *SalesForecastRepositorySQLC {
	r := &SalesForecastRepositorySQLC{q: q}
	if len(pool) > 0 {
		r.pool = pool[0]
	}
	return r
}
