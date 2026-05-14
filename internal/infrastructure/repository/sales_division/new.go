package sales_division

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type SalesDivisionRepositorySQLC struct {
	q *sqlc.Queries
}

func NewSalesDivisionRepositorySQLC(q *sqlc.Queries) *SalesDivisionRepositorySQLC {
	return &SalesDivisionRepositorySQLC{q: q}
}
