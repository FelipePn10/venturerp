package delivery_promise_params

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type DeliveryPromiseParamsRepositorySQLC struct {
	q *sqlc.Queries
}

func NewDeliveryPromiseParamsRepositorySQLC(q *sqlc.Queries) *DeliveryPromiseParamsRepositorySQLC {
	return &DeliveryPromiseParamsRepositorySQLC{q: q}
}
