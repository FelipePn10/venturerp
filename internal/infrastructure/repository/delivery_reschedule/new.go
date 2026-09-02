package delivery_reschedule

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliveryRescheduleRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewDeliveryRescheduleRepositorySQLC(q *sqlc.Queries, pools ...*pgxpool.Pool) *DeliveryRescheduleRepositorySQLC {
	r := &DeliveryRescheduleRepositorySQLC{q: q}
	if len(pools) > 0 {
		r.pool = pools[0]
	}
	return r
}
