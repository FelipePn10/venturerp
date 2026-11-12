package shipment_uc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type reservasAnotadas struct {
	reservas []*stockentity.StockReservation
}

func (r *reservasAnotadas) CreateReservation(_ context.Context, res *stockentity.StockReservation) (*stockentity.StockReservation, error) {
	r.reservas = append(r.reservas, res)
	return res, nil
}
func (r *reservasAnotadas) ListActiveReservations(context.Context) ([]*stockentity.StockReservation, error) {
	return nil, nil
}
func (r *reservasAnotadas) CancelReservation(context.Context, int64) error  { return nil }
func (r *reservasAnotadas) ConsumeReservation(context.Context, int64) error { return nil }

// romaneioComBaixa responde o que a NF-e já baixou.
type romaneioComBaixa struct {
	*fakeShipmentRepo
	baixa repository.BaixaPelaNFe
	nota  string
}

func (r *romaneioComBaixa) BaixaPelaNFe(context.Context, int64) (repository.BaixaPelaNFe, error) {
	return r.baixa, nil
}

func (r *romaneioComBaixa) UpdateStatus(ctx context.Context, code int64, status entity.ShipmentStatus, by *uuid.UUID, note string) error {
	r.nota = note
	return r.fakeShipmentRepo.UpdateStatus(ctx, code, status, by, note)
}

func linha(seq int, item, linhaPedido int64, qtd string) *entity.ShipmentItem {
	return &entity.ShipmentItem{Sequence: seq, ItemCode: item, SalesOrderItemCode: &linhaPedido, Quantity: decimal.RequireFromString(qtd)}
}

// Faturar antes de separar: a NF-e autorizada já baixou o estoque, e a
// separação pedia reserva de um saldo que não existe mais.
func TestSepararDepoisDaNotaReservaSoOQueAindaEstaNoEstoque(t *testing.T) {
	d := decimal.RequireFromString
	casos := []struct {
		nome      string
		baixa     repository.BaixaPelaNFe
		reservas  []string // quantidade reservada por linha, na ordem
		jaBaixado int
	}{
		{"sem nota", repository.BaixaPelaNFe{}, []string{"10", "4"}, 0},
		{"linha toda faturada", repository.BaixaPelaNFe{PorLinhaDoPedido: map[int64]decimal.Decimal{501: d("10")}}, []string{"4"}, 1},
		{"faturamento parcial", repository.BaixaPelaNFe{PorLinhaDoPedido: map[int64]decimal.Decimal{501: d("6"), 502: d("4")}}, []string{"4"}, 2},
		{"nota ligada ao romaneio", repository.BaixaPelaNFe{NotaVinculadaBaixou: true}, nil, 2},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			repo := &romaneioComBaixa{fakeShipmentRepo: &fakeShipmentRepo{getByCodeResp: &entity.Shipment{
				ID: 1, Code: 1001, Status: entity.ShipmentStatusOpen,
				Items: []*entity.ShipmentItem{linha(1, 7, 501, "10"), linha(2, 8, 502, "4")},
			}}, baixa: c.baixa}
			estoque := &reservasAnotadas{}
			uc := &ShipmentUseCase{Repo: repo, Stock: estoque}
			if err := uc.Separate(context.Background(), 1001, uuid.New()); err != nil {
				t.Fatal(err)
			}
			if len(estoque.reservas) != len(c.reservas) {
				t.Fatalf("reservas = %d, esperado %d", len(estoque.reservas), len(c.reservas))
			}
			for i, q := range c.reservas {
				if !estoque.reservas[i].Quantity.Equal(d(q)) {
					t.Fatalf("reserva %d = %s, esperado %s", i, estoque.reservas[i].Quantity, q)
				}
			}
			if repo.lastStatus != entity.ShipmentStatusSeparated {
				t.Fatalf("status = %s", repo.lastStatus)
			}
			if c.jaBaixado > 0 && repo.nota == "separação" {
				t.Fatalf("a separação não registrou as linhas já baixadas pela nota")
			}
		})
	}
}

// Duas linhas do romaneio na mesma linha do pedido: o faturado cobre a
// primeira e só o resto da segunda é reservado.
func TestFaturadoDaLinhaDoPedidoSeRepartePelasLinhasDoRomaneio(t *testing.T) {
	ship := &entity.Shipment{Code: 1, Items: []*entity.ShipmentItem{linha(1, 7, 501, "5"), linha(2, 7, 501, "5")}}
	estoque := &reservasAnotadas{}
	n, err := reserveStock(context.Background(), estoque, ship, uuid.New(),
		repository.BaixaPelaNFe{PorLinhaDoPedido: map[int64]decimal.Decimal{501: decimal.NewFromInt(7)}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || len(estoque.reservas) != 1 || !estoque.reservas[0].Quantity.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("linhas sem reserva=%d reservas=%+v", n, estoque.reservas)
	}
}
