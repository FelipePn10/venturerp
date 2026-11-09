package structure_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

// conversor de 1 UN = 85 KG: o fator KG→UN só cabe arredondado (0,0117647059).
type conversor85 struct{ exato bool }

func (conversor85) Factor(context.Context, int64, string, string) (decimal.Decimal, bool, error) {
	return decimal.NewFromInt(1).DivRound(decimal.NewFromInt(85), 10), true, nil
}
func (conversor85) ConvertQuantity(context.Context, int64, decimal.Decimal, string, string) (decimal.Decimal, bool, error) {
	return decimal.Zero, false, nil
}
func (conversor85) ConvertUnitPrice(context.Context, int64, decimal.Decimal, string, string) (decimal.Decimal, bool, error) {
	return decimal.Zero, false, nil
}

type conversor85Exato struct{ conversor85 }

func (conversor85Exato) ConvertQuantityExact(_ context.Context, _ int64, q decimal.Decimal, _, _ string) (decimal.Decimal, bool, error) {
	return q.DivRound(decimal.NewFromInt(85), 10), true, nil
}

func TestQuantidadeDaEstruturaPelaFracaoExata(t *testing.T) {
	ctx := context.Background()
	q, f, err := resolverUnidadeDeEstoque(ctx, conversor85Exato{}, 1, "CHAPA", decimal.RequireFromString("42.5"), "KG", "UN")
	if err != nil {
		t.Fatal(err)
	}
	if !q.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("42,5 KG de uma chapa de 85 KG = 0,5 UN exato, deu %s", q)
	}
	if f.String() != "0.0117647059" {
		t.Fatalf("o fator gravado continua o da conversão: %s", f)
	}
	// Conversor sem o método exato (fakes, integrações): cai no fator, como antes.
	q, _, err = resolverUnidadeDeEstoque(ctx, conversor85{}, 1, "CHAPA", decimal.RequireFromString("42.5"), "KG", "UN")
	if err != nil || q.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("sem o método exato o resultado é o do fator arredondado: %v %s", err, q)
	}
}
