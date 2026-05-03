package industrial_calendar

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type IndustrialCalendarRepositorySQLC struct {
	q *sqlc.Queries
}

func NewIndustrialCalendarRepositorySQLC(q *sqlc.Queries) *IndustrialCalendarRepositorySQLC {
	return &IndustrialCalendarRepositorySQLC{q: q}
}
