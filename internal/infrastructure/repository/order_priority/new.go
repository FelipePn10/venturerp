package order_priority

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type OrderPriorityRepositorySQLC struct {
	q *sqlc.Queries
}

func NewOrderPriorityRepositorySQLC(q *sqlc.Queries) *OrderPriorityRepositorySQLC {
	return &OrderPriorityRepositorySQLC{q: q}
}
