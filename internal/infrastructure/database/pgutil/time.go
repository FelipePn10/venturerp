package pgutil

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func ToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  t,
		Valid: true,
	}
}

func ToPgDatePtr(t *time.Time) *time.Time {
	return t
}

func FromPgDate(v pgtype.Date) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}

func FromPgDatePtr(v *time.Time) *time.Time {
	return v
}

func ToPgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: true,
	}
}

func ToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  t,
		Valid: true,
	}
}

func FromPgTimestamp(v pgtype.Timestamp) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}

func FromPgTimestamptz(v pgtype.Timestamptz) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}
