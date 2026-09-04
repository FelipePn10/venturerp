package item_calendar_promise

import (
	"context"
	"database/sql"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *ItemCalendarPromiseRepositorySQLC) UpsertDay(ctx context.Context, c *entity.ItemCalendarPromise) (*entity.ItemCalendarPromise, error) {
	row, err := r.q.UpsertItemCalendarDay(ctx, sqlc.UpsertItemCalendarDayParams{
		ItemCode:    c.ItemCode,
		Mask:        c.Mask,
		Year:        int32(c.Year),
		Month:       int32(c.Month),
		Day:         int32(c.Day),
		IsWorkday:   c.IsWorkday,
		Description: pgutil.ToPgTextFromPtr(c.Description),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting item calendar day: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *ItemCalendarPromiseRepositorySQLC) GetDay(ctx context.Context, itemCode int64, mask string, year, month, day int) (*entity.ItemCalendarPromise, error) {
	row, err := r.q.GetItemCalendarDay(ctx, sqlc.GetItemCalendarDayParams{
		ItemCode: itemCode,
		Mask:     mask,
		Year:     int32(year),
		Month:    int32(month),
		Day:      int32(day),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("dia do calendário do item não encontrado")
		}
		return nil, fmt.Errorf("fetching item calendar day: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *ItemCalendarPromiseRepositorySQLC) GetWorkdaysInMonth(ctx context.Context, itemCode int64, mask string, year, month int) ([]*entity.ItemCalendarPromise, error) {
	rows, err := r.q.GetItemWorkdaysInMonth(ctx, sqlc.GetItemWorkdaysInMonthParams{
		ItemCode: itemCode,
		Mask:     mask,
		Year:     int32(year),
		Month:    int32(month),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching item workdays: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *ItemCalendarPromiseRepositorySQLC) ListMonth(ctx context.Context, itemCode int64, mask string, year, month int) ([]*entity.ItemCalendarPromise, error) {
	rows, err := r.q.ListItemCalendarMonth(ctx, sqlc.ListItemCalendarMonthParams{
		ItemCode: itemCode,
		Mask:     mask,
		Year:     int32(year),
		Month:    int32(month),
	})
	if err != nil {
		return nil, fmt.Errorf("listing item calendar month: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *ItemCalendarPromiseRepositorySQLC) DeleteDay(ctx context.Context, itemCode int64, mask string, year, month, day int) error {
	return r.q.DeleteItemCalendarDay(ctx, sqlc.DeleteItemCalendarDayParams{
		ItemCode: itemCode,
		Mask:     mask,
		Year:     int32(year),
		Month:    int32(month),
		Day:      int32(day),
	})
}

func rowToEntity(row sqlc.ItemCalendarPromise) *entity.ItemCalendarPromise {
	e := &entity.ItemCalendarPromise{
		ID:        row.ID,
		ItemCode:  row.ItemCode,
		Mask:      row.Mask,
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

func rowsToEntities(rows []sqlc.ItemCalendarPromise) []*entity.ItemCalendarPromise {
	out := make([]*entity.ItemCalendarPromise, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}
