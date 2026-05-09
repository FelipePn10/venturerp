package machine

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type MachineRepositorySQLC struct {
	q *sqlc.Queries
}

func NewMachineRepositorySQLC(q *sqlc.Queries) *MachineRepositorySQLC {
	return &MachineRepositorySQLC{q: q}
}
