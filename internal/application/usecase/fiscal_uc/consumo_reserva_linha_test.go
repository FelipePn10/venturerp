package fiscal_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type reservasComSaldo struct {
	stockrepo.StockRepository
	ativas   []*stockentity.StockReservation
	consumos map[int64]decimal.Decimal
}

func (r *reservasComSaldo) ListActiveReservations(context.Context) ([]*stockentity.StockReservation, error) {
	return r.ativas, nil
}

func (r *reservasComSaldo) ConsumirReservaParcial(_ context.Context, id int64, q decimal.Decimal) (decimal.Decimal, error) {
	for _, a := range r.ativas {
		if a.ID == id {
			c := decimal.Min(q, a.Quantity.Sub(r.consumos[id]))
			if !c.IsPositive() {
				return decimal.Zero, nil
			}
			r.consumos[id] = r.consumos[id].Add(c)
			return c, nil
		}
	}
	return decimal.Zero, nil
}

// Produto em duas linhas: a nota da linha 502 consome a reserva da 502, não a
// da 501 (que vinha primeiro e ficava presa).
func TestNotaConsomePrimeiroAReservaDaLinhaFaturada(t *testing.T) {
	l501, l502, prod, pedido := int64(501), int64(502), int64(1001), int64(7)
	dez := decimal.NewFromInt(10)
	estoque := &reservasComSaldo{consumos: map[int64]decimal.Decimal{}, ativas: []*stockentity.StockReservation{
		{ID: 1, ItemCode: prod, Quantity: dez, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, ReferenceItemCode: &l501},
		{ID: 2, ItemCode: prod, Quantity: dez, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, ReferenceItemCode: &l502},
		{ID: 3, ItemCode: prod, Quantity: dez, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, ReferenceItemCode: &prod},
	}}
	itens := []*entity.FiscalExitItem{{ItemCode: &prod, SalesOrderItemCode: &l502, Quantity: decimal.NewFromInt(15)}}
	exit := &entity.FiscalExit{SalesOrderCode: &pedido}
	if err := consumirReservasDoPedido(context.Background(), estoque, exit, faturadoDaNota(itens, func(it *entity.FiscalExitItem) decimal.Decimal { return it.Quantity })); err != nil {
		t.Fatal(err)
	}
	// 10 da linha 502, os 5 que passam saem da reserva sem linha; a da 501 fica.
	if !estoque.consumos[2].Equal(dez) || !estoque.consumos[3].Equal(decimal.NewFromInt(5)) || !estoque.consumos[1].IsZero() {
		t.Fatalf("consumos = %v", estoque.consumos)
	}
}

type romaneiosFixos []int64

func (r romaneiosFixos) RomaneiosDaNota(context.Context, int64) ([]int64, error) { return r, nil }
func (romaneiosFixos) ReabrirPelaNotaCancelada(context.Context, int64) error     { return nil }

// Bug L: a NF-e baixava o estoque e a reserva da separação do romaneio ficava
// ativa até o despacho (saldo 0 com 2 reservados). Consome só a do romaneio da
// nota e só até o faturado.
func TestNotaConsomeAReservaDoRomaneio(t *testing.T) {
	prod := int64(1001)
	d := decimal.NewFromInt
	estoque := &reservasComSaldo{consumos: map[int64]decimal.Decimal{}, ativas: []*stockentity.StockReservation{
		{ID: 1, ItemCode: prod, Quantity: d(2), ReferenceType: referenciaRomaneio, ReferenceCode: 40},
		{ID: 2, ItemCode: prod, Quantity: d(5), ReferenceType: referenciaRomaneio, ReferenceCode: 41}, // outro romaneio
		{ID: 3, ItemCode: prod, Quantity: d(3), ReferenceType: referenciaRomaneio, ReferenceCode: 40},
	}}
	itens := []*entity.FiscalExitItem{{ItemCode: &prod, Quantity: d(4)}}
	r := EfeitosRepos{Estoque: estoque, Romaneios: romaneiosFixos{40}}
	if err := consumirReservasDosRomaneios(context.Background(), r, 9, itens, func(it *entity.FiscalExitItem) decimal.Decimal { return it.Quantity }); err != nil {
		t.Fatal(err)
	}
	if !estoque.consumos[1].Equal(d(2)) || !estoque.consumos[3].Equal(d(2)) || !estoque.consumos[2].IsZero() {
		t.Fatalf("consumos = %v", estoque.consumos)
	}
}
