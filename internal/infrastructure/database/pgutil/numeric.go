package pgutil

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
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

func ToPgNumericFromFloat64Ptr(v *float64) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	return ToPgNumericFromFloat64(*v)
}

// ToPgNumericFromDecimal grava o decimal sem passar por float64, preservando a
// escala exata digitada.
func ToPgNumericFromDecimal(v decimal.Decimal) pgtype.Numeric {
	return pgtype.Numeric{Int: v.Coefficient(), Exp: v.Exponent(), Valid: true}
}

// ToPgNumericFromDecimalPtr devolve NULL para nil.
func ToPgNumericFromDecimalPtr(v *decimal.Decimal) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	return ToPgNumericFromDecimal(*v)
}

// FromPgNumericToDecimal lê o numeric exato; NULL, NaN e infinito viram zero.
func FromPgNumericToDecimal(v pgtype.Numeric) decimal.Decimal {
	if !v.Valid || v.Int == nil || v.NaN || v.InfinityModifier != pgtype.Finite {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(v.Int, v.Exp)
}

// FromPgNumericToDecimalPtr devolve nil para NULL.
func FromPgNumericToDecimalPtr(v pgtype.Numeric) *decimal.Decimal {
	if !v.Valid {
		return nil
	}
	d := FromPgNumericToDecimal(v)
	return &d
}
