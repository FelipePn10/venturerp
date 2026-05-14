package sales_forecast

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type SalesForecastRepositorySQLC struct {
	q *sqlc.Queries
}

func NewSalesForecastRepositorySQLC(q *sqlc.Queries) *SalesForecastRepositorySQLC {
	return &SalesForecastRepositorySQLC{q: q}
}
