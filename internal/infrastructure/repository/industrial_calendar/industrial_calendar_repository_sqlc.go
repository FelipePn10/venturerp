package industrial_calendar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

func (r *IndustrialCalendarRepositorySQLC) CreateDay(
	ctx context.Context,
	c *entity.IndustrialCalendar,
) (*entity.IndustrialCalendar, error) {

	row, err := r.q.CreateCalendarDay(ctx, sqlc.CreateCalendarDayParams{
		Year:        int32(c.Year),
		Month:       int32(c.Month),
		Day:         int32(c.Day),
		IsWorkday:   c.IsWorkday,
		Description: pgutil.ToPgTextFromPtr(c.Description),
	})
	if err != nil {
		return nil, fmt.Errorf("creating calendar day: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndustrialCalendarRepositorySQLC) GetDay(
	ctx context.Context,
	year, month, day int,
) (*entity.IndustrialCalendar, error) {

	row, err := r.q.GetCalendarDay(ctx, sqlc.GetCalendarDayParams{
		Year:  int32(year),
		Month: int32(month),
		Day:   int32(day),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("calendar day not found")
		}
		return nil, fmt.Errorf("fetching calendar day: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndustrialCalendarRepositorySQLC) GetWorkdaysInMonth(
	ctx context.Context,
	year, month int,
) ([]*entity.IndustrialCalendar, error) {

	rows, err := r.q.GetWorkdaysInMonth(ctx, sqlc.GetWorkdaysInMonthParams{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching workdays: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *IndustrialCalendarRepositorySQLC) IsWorkday(
	ctx context.Context,
	year, month, day int,
) (bool, error) {

	result, err := r.q.IsWorkday(ctx, sqlc.IsWorkdayParams{
		Year:  int32(year),
		Month: int32(month),
		Day:   int32(day),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("checking workday: %w", err)
	}

	return result, nil
}

func (r *IndustrialCalendarRepositorySQLC) GetNextWorkday(
	ctx context.Context,
	year, month, day int,
) (time.Time, error) {

	row, err := r.q.GetNextWorkday(ctx, sqlc.GetNextWorkdayParams{
		Year:  int32(year),
		Month: int32(month),
		Day:   int32(day),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("no next workday found")
		}
		return time.Time{}, fmt.Errorf("fetching next workday: %w", err)
	}

	return time.Date(
		int(row.Year),
		time.Month(row.Month),
		int(row.Day),
		0, 0, 0, 0,
		time.UTC,
	), nil
}

func (r *IndustrialCalendarRepositorySQLC) ListMonth(
	ctx context.Context,
	year, month int,
) ([]*entity.IndustrialCalendar, error) {

	rows, err := r.q.ListCalendarMonth(ctx, sqlc.ListCalendarMonthParams{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		return nil, fmt.Errorf("listing calendar month: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *IndustrialCalendarRepositorySQLC) DeleteDay(
	ctx context.Context,
	year, month, day int,
) error {

	return r.q.DeleteCalendarDay(ctx, sqlc.DeleteCalendarDayParams{
		Year:  int32(year),
		Month: int32(month),
		Day:   int32(day),
	})
}

func rowToEntity(row sqlc.IndustrialCalendar) *entity.IndustrialCalendar {
	e := &entity.IndustrialCalendar{
		Year:      int(row.Year),
		Month:     int(row.Month),
		Day:       int(row.Day),
		IsWorkday: row.IsWorkday,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
	}

	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}

	return e
}

func rowsToEntities(
	rows []sqlc.IndustrialCalendar,
) []*entity.IndustrialCalendar {

	out := make([]*entity.IndustrialCalendar, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}
