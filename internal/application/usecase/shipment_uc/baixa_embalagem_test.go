package shipment_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
)

type regrasDaEmbalagem struct {
	regrasFixas
	saldo decimal.Decimal
}

func (r regrasDaEmbalagem) Embalagens(context.Context, []int64) (map[int64]EmbalagemDoItem, error) {
	return map[int64]EmbalagemDoItem{1: {ItemCode: 9, WarehouseID: 2}}, nil
}
func (r regrasDaEmbalagem) SaldoDaEmbalagem(context.Context, EmbalagemDoItem) (decimal.Decimal, error) {
	return r.saldo, nil
}

func TestBaixaDaEmbalagemContaOConferidoENaoSeguraODespacho(t *testing.T) {
	ship := &entity.Shipment{Code: 1, Items: []*entity.ShipmentItem{{ItemCode: 1, Quantity: decimal.NewFromInt(10), ConferredQty: decimal.NewFromInt(8), IsConferred: true}}}
	uc := &ShipmentUseCase{Regras: regrasDaEmbalagem{saldo: decimal.NewFromInt(100)}}
	baixas, falta, err := uc.baixasDeEmbalagem(context.Background(), ship, uuid.New())
	if err != nil || falta != "" || len(baixas) != 1 || !baixas[0].Quantity.Equal(decimal.NewFromInt(8)) {
		t.Fatalf("saem 8 caixas, as conferidas: %+v %q %v", baixas, falta, err)
	}
	uc = &ShipmentUseCase{Regras: regrasDaEmbalagem{saldo: decimal.NewFromInt(3)}}
	baixas, falta, err = uc.baixasDeEmbalagem(context.Background(), ship, uuid.New())
	if err != nil || len(baixas) != 1 || !baixas[0].Quantity.Equal(decimal.NewFromInt(3)) || !strings.Contains(falta, "faltaram 5") {
		t.Fatalf("baixa o saldo e registra a falta: %+v %q %v", baixas, falta, err)
	}
	uc = &ShipmentUseCase{Regras: regrasDaEmbalagem{}}
	baixas, falta, err = uc.baixasDeEmbalagem(context.Background(), ship, uuid.New())
	if err != nil || len(baixas) != 0 || !strings.Contains(falta, "faltaram 8") {
		t.Fatalf("sem saldo nenhum, não há baixa: %+v %q %v", baixas, falta, err)
	}
}
