package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
)

type IndustrialCalendarRepository interface {
	CreateDay(ctx context.Context, c *entity.IndustrialCalendar) (*entity.IndustrialCalendar, error)
	GetDay(ctx context.Context, year, month, day int) (*entity.IndustrialCalendar, error)
	GetWorkdaysInMonth(ctx context.Context, year, month int) ([]*entity.IndustrialCalendar, error)
	IsWorkday(ctx context.Context, year, month, day int) (bool, error)
	GetNextWorkday(ctx context.Context, year, month, day int) (time.Time, error)
	ListMonth(ctx context.Context, year, month int) ([]*entity.IndustrialCalendar, error)
	DeleteDay(ctx context.Context, year, month, day int) error

	// SubtractWorkdays moves `days` working days backward from `from` using the
	// subtract_workdays PostgreSQL function — single round-trip to the database.
	SubtractWorkdays(ctx context.Context, from time.Time, days int) (time.Time, error)
}
