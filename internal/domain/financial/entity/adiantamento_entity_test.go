package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestAdiantamentoSaldo(t *testing.T) {
	a := &Adiantamento{ValorOriginal: decimal.NewFromInt(100), ValorUtilizado: decimal.NewFromInt(30), Status: AdiantamentoStatusParcial}
	if !a.Saldo().Equal(decimal.NewFromInt(70)) {
		t.Fatalf("saldo = %s, want 70", a.Saldo())
	}
	a.Status = AdiantamentoStatusCancelado
	if !a.Saldo().IsZero() {
		t.Fatalf("adiantamento cancelado não pode ter saldo a aplicar: %s", a.Saldo())
	}
}
