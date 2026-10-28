package entrada

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

func linhaComCreditos() *entity.FiscalEntryItem {
	return &entity.FiscalEntryItem{GeraCreditoICMS: true, GeraCreditoIPI: true, GeraCreditoPIS: true, GeraCreditoCOFINS: true}
}

func TestCreditosDoRegime(t *testing.T) {
	it := linhaComCreditos()
	if got := CreditosDoRegime(engine.RegimeSimples, it); len(got) != 4 || it.GeraCreditoICMS || it.GeraCreditoIPI || it.GeraCreditoPIS || it.GeraCreditoCOFINS {
		t.Fatalf("Simples: tirados %v, linha %+v — quer nenhum crédito", got, it)
	}
	it = linhaComCreditos()
	if got := CreditosDoRegime(engine.RegimePresumido, it); len(got) != 2 || !it.GeraCreditoICMS || !it.GeraCreditoIPI || it.GeraCreditoPIS || it.GeraCreditoCOFINS {
		t.Fatalf("Presumido: tirados %v, linha %+v — quer só ICMS e IPI", got, it)
	}
	it = linhaComCreditos()
	if got := CreditosDoRegime(engine.RegimeReal, it); len(got) != 0 || !it.GeraCreditoPIS {
		t.Fatalf("Real: tirados %v — quer todos os créditos", got)
	}
}

// R$ 1.000 de SP para o PR (ICMS da origem 12% = 120), interna de 19,5%:
// base = 880 / 0,805 = 1.093,17 e DIFAL = 1.093,17 × 19,5% − 120 = 93,17.
func TestDifalUsoConsumoBaseDupla(t *testing.T) {
	base, difal := DifalUsoConsumo(decimal.NewFromInt(1000), decimal.NewFromInt(120), decimal.NewFromFloat(0.195))
	if !base.Equal(decimal.RequireFromString("1093.17")) || !difal.Equal(decimal.RequireFromString("93.17")) {
		t.Fatalf("base %s DIFAL %s, quer 1093.17 e 93.17", base, difal)
	}
	if _, d := DifalUsoConsumo(decimal.NewFromInt(1000), decimal.NewFromInt(200), decimal.NewFromFloat(0.12)); !d.IsZero() {
		t.Fatalf("origem acima da interna não gera DIFAL, deu %s", d)
	}
}

func TestCFOPDeUsoConsumoInterestadual(t *testing.T) {
	for cfop, quer := range map[string]bool{"2556": true, "2.551": true, "2407": true, "1556": false, "2102": false} {
		if got := CFOPDeUsoConsumoInterestadual(cfop); got != quer {
			t.Errorf("%s: %v, quer %v", cfop, got, quer)
		}
	}
}
