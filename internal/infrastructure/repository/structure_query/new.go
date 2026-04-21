package structure_query

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type StructureQueryRepositorySQLC struct {
	q *sqlc.Queries
}

func NewStructureQueryRepository(q *sqlc.Queries) *StructureQueryRepositorySQLC {
	return &StructureQueryRepositorySQLC{q: q}
}
