package pgutil

import "github.com/jackc/pgx/v5/pgtype"

func ToPgBool(v bool) pgtype.Bool {
	return pgtype.Bool{
		Bool:  v,
		Valid: true,
	}
}
