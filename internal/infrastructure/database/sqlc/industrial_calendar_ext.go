package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const subtractWorkdaysSQL = `SELECT subtract_workdays($1::date, $2::int)`

// SubtractWorkdays calls the PostgreSQL function created in migration 000080.
// It returns the date that is `days` working days before `from`, consulting the
// industrial_calendar table inside the DB function — a single round-trip.
func (q *Queries) SubtractWorkdays(ctx context.Context, from time.Time, days int) (time.Time, error) {
	fromDate := pgtype.Date{Time: from.UTC().Truncate(24 * time.Hour), Valid: true}
	var result pgtype.Date
	err := q.db.QueryRow(ctx, subtractWorkdaysSQL, fromDate, int32(days)).Scan(&result)
	if err != nil {
		return time.Time{}, err
	}
	return result.Time, nil
}
