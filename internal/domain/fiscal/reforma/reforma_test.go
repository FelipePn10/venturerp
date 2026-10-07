package reforma

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestCalcular2026(t *testing.T) {
	g, err := Calcular(2026, d("1000"), d("180"), d("16.50"), d("76"))
	if err != nil {
		t.Fatal(err)
	}
	// base 1000 − 180 − 16,50 − 76 = 727,50; CBS 0,9% = 6,55 (6,5475); IBS UF 0,1% = 0,73 (0,7275).
	if !g.Base.Equal(d("727.5")) || !g.CBS.Equal(d("6.55")) || !g.IBSUF.Equal(d("0.73")) || !g.IBSMun.IsZero() || !g.IBS.Equal(d("0.73")) {
		t.Fatalf("grupo = %+v", g)
	}
	if g.CST != "000" || g.ClassTrib != "000001" {
		t.Fatalf("CST/cClassTrib = %s/%s", g.CST, g.ClassTrib)
	}
	if g, _ := Calcular(2026, d("10"), d("20"), d("0"), d("0")); !g.Base.IsZero() {
		t.Fatalf("base negativa deveria zerar: %s", g.Base)
	}
}

func TestAnoSemAliquota(t *testing.T) {
	if _, err := Calcular(2027, d("10"), d("0"), d("0"), d("0")); err == nil {
		t.Fatal("2027 sem tabela deveria falhar (e não emitir nota com alíquota errada)")
	}
}
