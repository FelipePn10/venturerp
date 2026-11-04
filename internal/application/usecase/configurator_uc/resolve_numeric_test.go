package configurator_uc

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// 0,3 é múltiplo de 0,1. Em float, math.Mod(0.3, 0.1) dava 0,0999… e a
// resposta válida era recusada.
func TestResolveNumericMultiploDecimal(t *testing.T) {
	c := sqlc.DBCfgCharacteristic{Code: "ESP", NumMultiple: pgutil.ToPgNumericFromDecimal(decimal.RequireFromString("0.1"))}
	if _, _, err := resolveNumeric(c, "0.3"); err != nil {
		t.Fatalf("0,3 deveria ser múltiplo de 0,1: %v", err)
	}
	if _, _, err := resolveNumeric(c, "0.35"); err == nil {
		t.Fatal("0,35 não é múltiplo de 0,1")
	}
	if _, _, err := resolveNumeric(c, "abc"); err == nil {
		t.Fatal("texto não é número")
	}
}
