package structure

import (
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemStructureRepositorySQLC struct {
	q *sqlc.Queries
	// pool serve ao histórico da estrutura, que não passa por sqlc para não
	// obrigar a regerar o pacote a cada campo novo do instantâneo.
	pool *pgxpool.Pool
}

func NewItemStructureRepository(q *sqlc.Queries) *ItemStructureRepositorySQLC {
	return &ItemStructureRepositorySQLC{q: q}
}

// WithHistory liga o registro do histórico. Sem o pool o repositório continua
// funcionando; apenas não grava a trilha de auditoria.
func (r *ItemStructureRepositorySQLC) WithHistory(pool *pgxpool.Pool) *ItemStructureRepositorySQLC {
	r.pool = pool
	return r
}
