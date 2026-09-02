package machine

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MachineRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewMachineRepositorySQLC(q *sqlc.Queries, pools ...*pgxpool.Pool) *MachineRepositorySQLC {
	repository := &MachineRepositorySQLC{q: q}
	if len(pools) > 0 {
		repository.pool = pools[0]
	}
	return repository
}
