package delivery_reschedule

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type DeliveryRescheduleRepositorySQLC struct {
	q *sqlc.Queries
}

func NewDeliveryRescheduleRepositorySQLC(q *sqlc.Queries) *DeliveryRescheduleRepositorySQLC {
	return &DeliveryRescheduleRepositorySQLC{q: q}
}
