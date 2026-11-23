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

// reservasDoPedidoAtivas devolve as reservas que o pedido já fez.
type reservasDoPedidoAtivas struct {
	reservasAnotadas
	ativas []*stockentity.StockReservation
}

func (r *reservasDoPedidoAtivas) ListActiveReservations(context.Context) ([]*stockentity.StockReservation, error) {
	return r.ativas, nil
}

// Bug J: o romaneio do pedido reservava tudo de novo sobre o saldo que o
// pedido já tinha reservado, e a separação recusava por falta de disponível.
func TestSepararAproveitaAReservaDoPedido(t *testing.T) {
	d := decimal.RequireFromString
	pedido := int64(77)
	l501, l502 := int64(501), int64(502)
	estoque := &reservasDoPedidoAtivas{ativas: []*stockentity.StockReservation{
		{ItemCode: 7, Mask: "M1", WarehouseID: 3, Quantity: d("10"), ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, ReferenceItemCode: &l501},
		{ItemCode: 8, WarehouseID: 3, Quantity: d("1"), ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, ReferenceItemCode: &l502},
		// Reserva de outro pedido não conta.
		{ItemCode: 8, WarehouseID: 3, Quantity: d("50"), ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 99, ReferenceItemCode: &l502},
	}}
	ship := &entity.Shipment{Code: 1, SalesOrderCode: &pedido, Items: []*entity.ShipmentItem{linha(1, 7, 501, "10"), linha(2, 8, 502, "4")}}
	if _, err := reserveStock(context.Background(), estoque, ship, uuid.New(), repository.BaixaPelaNFe{}); err != nil {
		t.Fatal(err)
	}
	if len(estoque.reservas) != 1 {
		t.Fatalf("reservas novas = %+v, esperado só o que falta da linha 502", estoque.reservas)
	}
	r := estoque.reservas[0]
	if !r.Quantity.Equal(d("3")) || r.ReferenceType != stockentity.ReferenceTypeSalesOrder || r.ReferenceCode != pedido ||
		r.ReferenceItemCode == nil || *r.ReferenceItemCode != 502 || r.WarehouseID != 3 {
		t.Fatalf("reserva = %+v", r)
	}
}

// O romaneio montado do pedido leva a linha, o almoxarifado e só o pendente.
func TestAutoFillLevaLinhaAlmoxarifadoEPendente(t *testing.T) {
	d := decimal.RequireFromString
	almox := int64(4)
	repo := &fakeShipmentRepo{nextCode: 1, getByCodeResp: &entity.Shipment{Code: 1}}
	sales := &fakeSalesReader{resp: &SalesOrderHeader{Code: 100, TotalVolumes: 1, Items: []SalesOrderItemHeader{
		{LineCode: 7, ItemCode: 1, WarehouseID: &almox, RequestedQty: d("10"), PendingQty: d("3")},
		{LineCode: 8, ItemCode: 2, RequestedQty: d("5"), PendingQty: d("0")},
	}}}
	if _, err := newAutoFillUC(repo, sales, nil, nil).AutoFillFromSalesOrder(context.Background(), 100, uuid.New()); err != nil {
		t.Fatal(err)
	}
	it := repo.addedItem
	if it == nil || it.SalesOrderItemCode == nil || *it.SalesOrderItemCode != 7 || it.WarehouseID == nil || *it.WarehouseID != 4 || !it.Quantity.Equal(d("3")) {
		t.Fatalf("item do romaneio = %+v", it)
	}
}

// Item configurado sem reserva do pedido: a reserva completada leva a máscara da linha.
func TestReservaCompletadaLevaAMascaraDaLinha(t *testing.T) {
	pedido := int64(77)
	almox := int64(3)
	estoque := &reservasDoPedidoAtivas{}
	it := linha(1, 7, 501, "2")
	it.WarehouseID = &almox
	ship := &entity.Shipment{Code: 1, SalesOrderCode: &pedido, Items: []*entity.ShipmentItem{it}}
	if _, err := reserveStock(context.Background(), estoque, ship, uuid.New(), repository.BaixaPelaNFe{}, map[int64]string{501: "AZUL-2M"}); err != nil {
		t.Fatal(err)
	}
	if len(estoque.reservas) != 1 || estoque.reservas[0].Mask != "AZUL-2M" {
		t.Fatalf("reserva: %+v", estoque.reservas)
	}
}

type regrasComDeposito struct{ padrao map[int64]int64 }

func (regrasComDeposito) ItensQueExigemVolume(context.Context, []int64) ([]string, error) {
	return nil, nil
}
func (regrasComDeposito) VolumesPorUnidade(context.Context, []int64) (map[int64]decimal.Decimal, error) {
	return nil, nil
}
func (regrasComDeposito) Embalagens(context.Context, []int64) (map[int64]EmbalagemDoItem, error) {
	return nil, nil
}
func (regrasComDeposito) SaldoDaEmbalagem(context.Context, EmbalagemDoItem) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (r regrasComDeposito) DepositosPadrao(context.Context, []int64) (map[int64]int64, error) {
	return r.padrao, nil
}

// Linha do romaneio sem depósito separa no depósito padrão do item (a regra da
// NF-e); a que já tem depósito fica com o dela. Antes a reserva ia para o 0.
func TestSeparacaoUsaODepositoPadraoDoItem(t *testing.T) {
	proprio := int64(9)
	semDep, comDep := linha(1, 7, 501, "2"), linha(2, 8, 502, "1")
	comDep.WarehouseID = &proprio
	ship := &entity.Shipment{Code: 1, Items: []*entity.ShipmentItem{semDep, comDep}}
	uc := &ShipmentUseCase{Regras: regrasComDeposito{padrao: map[int64]int64{7: 2, 8: 3}}}
	if err := uc.completarDepositos(context.Background(), ship); err != nil {
		t.Fatal(err)
	}
	if semDep.WarehouseID == nil || *semDep.WarehouseID != 2 || *comDep.WarehouseID != 9 {
		t.Fatalf("depósitos: %v / %v", semDep.WarehouseID, *comDep.WarehouseID)
	}
}
