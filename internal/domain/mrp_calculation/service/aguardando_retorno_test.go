package service

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/shopspring/decimal"
)

func TestSomarAguardandoRetorno(t *testing.T) {
	const comSaldo, semSaldo, zerado, negativo = int64(1), int64(2), int64(3), int64(4)
	original := &entity.StockSnapshot{ItemCode: comSaldo, Quantity: decimal.NewFromInt(5)}
	snapshots := map[int64]*entity.StockSnapshot{comSaldo: original}

	somarAguardandoRetorno(map[int64]decimal.Decimal{
		comSaldo: decimal.NewFromInt(6),
		semSaldo: decimal.RequireFromString("2.5"),
		zerado:   decimal.Zero,
		negativo: decimal.NewFromInt(-3),
	}, snapshots)

	if got := snapshots[comSaldo].Quantity.String(); got != "11" {
		t.Fatalf("item com saldo = %s, esperado 11", got)
	}
	if original.Quantity.String() != "5" {
		t.Fatalf("o snapshot original foi alterado: %s", original.Quantity)
	}
	if got := snapshots[semSaldo]; got == nil || got.ItemCode != semSaldo || got.Quantity.String() != "2.5" {
		t.Fatalf("item sem saldo = %+v, esperado 2.5", got)
	}
	if _, ok := snapshots[zerado]; ok {
		t.Fatal("quantidade zerada não deve criar snapshot")
	}
	if _, ok := snapshots[negativo]; ok {
		t.Fatal("quantidade negativa não deve criar snapshot")
	}
}
