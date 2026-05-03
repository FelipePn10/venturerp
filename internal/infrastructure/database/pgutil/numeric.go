package pgutil

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

func ToPgNumericFromString(s string) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(s)
	return n
}

func ToPgNumericFromFloat64(v float64) pgtype.Numeric {
	return ToPgNumericFromString(
		strconv.FormatFloat(v, 'f', -1, 64),
	)
}

func FromPgNumericToString(v pgtype.Numeric) string {
	if !v.Valid {
		return ""
	}

	val, err := v.Value()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%v", val)
}

func FromPgNumericToFloat64(v pgtype.Numeric) float64 {
	s := FromPgNumericToString(v)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return f
}
