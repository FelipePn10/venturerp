package sales_order

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type SalesOrderRepositorySQLC struct {
	q *sqlc.Queries
}

func NewSalesOrderRepositorySQLC(q *sqlc.Queries) *SalesOrderRepositorySQLC {
	return &SalesOrderRepositorySQLC{q: q}
}
