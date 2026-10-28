package pgutil

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func TestDecimalIdaEVoltaSemPerda(t *testing.T) {
	// 0,1 + 0,2 em float64 dá 0,30000000000000004; no decimal tem de voltar exato.
	soma := decimal.RequireFromString("0.1").Add(decimal.RequireFromString("0.2"))
	for _, v := range []decimal.Decimal{soma, decimal.RequireFromString("15.530965"), decimal.RequireFromString("-0.000001"), decimal.Zero} {
		got := FromPgNumericToDecimal(ToPgNumericFromDecimal(v))
		if !got.Equal(v) {
			t.Fatalf("esperado %s, veio %s", v, got)
		}
	}
}

func TestDecimalNuloENaN(t *testing.T) {
	if FromPgNumericToDecimalPtr(pgtype.Numeric{}) != nil {
		t.Fatal("NULL deveria virar nil")
	}
	if ToPgNumericFromDecimalPtr(nil).Valid {
		t.Fatal("nil deveria virar NULL")
	}
	if !FromPgNumericToDecimal(pgtype.Numeric{NaN: true, Valid: true}).IsZero() {
		t.Fatal("NaN deveria virar zero")
	}
}
