package shipment_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

type regrasFixas map[int64]decimal.Decimal

func (r regrasFixas) ItensQueExigemVolume(context.Context, []int64) ([]string, error) {
	return nil, nil
}
func (r regrasFixas) VolumesPorUnidade(context.Context, []int64) (map[int64]decimal.Decimal, error) {
	return r, nil
}
func (r regrasFixas) SaldoDaEmbalagem(context.Context, EmbalagemDoItem) (decimal.Decimal, error) {
	return decimal.NewFromInt(1000000), nil
}

func (r regrasFixas) Embalagens(context.Context, []int64) (map[int64]EmbalagemDoItem, error) {
	return nil, nil
}

func TestVolumesDoPedidoVemDoCadastro(t *testing.T) {
	so := &SalesOrderHeader{Items: []SalesOrderItemHeader{
		{ItemCode: 1, RequestedQty: decimal.NewFromInt(2)},
		{ItemCode: 2, RequestedQty: decimal.RequireFromString("1.5")},
		{ItemCode: 3, RequestedQty: decimal.NewFromInt(10)},
	}}
	uc := &ShipmentAutoFillUseCase{Regras: regrasFixas{1: decimal.NewFromInt(3), 2: decimal.NewFromInt(2)}}
	total, err := uc.volumesDoPedido(context.Background(), so)
	if err != nil {
		t.Fatal(err)
	}
	if total != 9 { // 2×3 + 1,5×2; o item 3 não informa
		t.Fatalf("volumes = %d", total)
	}
	if total, _ = (&ShipmentAutoFillUseCase{}).volumesDoPedido(context.Background(), so); total != 1 {
		t.Fatalf("sem regra segue um volume: %d", total)
	}
}
