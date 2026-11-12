package sales_order_uc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// OrderStockReserver reserves available stock for each line of a confirmed sales
// order (available-to-promise). The reservation is capped at the currently
// available quantity, so a promise is never made beyond what exists. It is
// idempotent: an order that already holds active reservations is left untouched.
type OrderStockReserver struct {
	SalesRepo repository.SalesOrderRepository
	StockRepo stockrepo.StockRepository
}

// Reserve reserva o disponível de cada linha, com os repositórios da transação
// da confirmação. Qualquer falha desfaz a confirmação — antes era ignorada e o
// pedido ficava confirmado sem a reserva.
func (r *OrderStockReserver) Reserve(ctx context.Context, code int64, pedidos repository.SalesOrderRepository, estoque stockrepo.StockRepository) error {
	already, err := estoque.HasActiveReservationByReference(ctx, stockentity.ReferenceTypeSalesOrder, code)
	if err != nil {
		return fmt.Errorf("conferindo as reservas do pedido: %w", err)
	}
	if already {
		return nil
	}

	order, err := pedidos.GetByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo o pedido: %w", err)
	}
	items, err := pedidos.ListItems(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo os itens do pedido: %w", err)
	}

	for _, it := range items {
		if !it.IsActive || it.Status == entity.SalesOrderItemStatusCancelled || it.WarehouseCode == nil {
			continue
		}
		wanted := it.RequestedQty.Sub(it.AttendedQty).Sub(it.CancelledQty)
		if !wanted.IsPositive() {
			continue
		}

		if err := reservarDisponivel(ctx, estoque, order, it, wanted); err != nil {
			return err
		}
	}
	return nil
}

// reservarDisponivel reserva para a linha até qty, limitado ao disponível do
// depósito dela. Depósito que não aceita reserva deixa a linha sem reservar.
func reservarDisponivel(ctx context.Context, estoque stockrepo.StockRepository, order *entity.SalesOrder, it *entity.SalesOrderItem, qty decimal.Decimal) error {
	// Sem saldo cadastrado no depósito o disponível é zero; falha de leitura é erro.
	available := decimal.Zero
	bal, balErr := estoque.GetBalance(ctx, it.ItemCode, it.Mask, *it.WarehouseCode)
	if balErr != nil && !errorsuc.EhNaoEncontrado(balErr) {
		return fmt.Errorf("lendo o saldo do item %d: %w", it.ItemCode, balErr)
	}
	if balErr == nil {
		available = bal.AvailableQty
	}
	reserveQty := decimal.Min(qty, available)
	if !reserveQty.IsPositive() {
		return nil
	}

	// reference_item_code é a LINHA do pedido (como no romaneio e na
	// reprogramação da entrega); gravava o código do produto, e a
	// reprogramação e o cancelamento da linha não achavam a reserva.
	linha := it.Code
	res := &stockentity.StockReservation{
		ItemCode:          it.ItemCode,
		Mask:              it.Mask,
		WarehouseID:       *it.WarehouseCode,
		Quantity:          reserveQty,
		ReferenceType:     stockentity.ReferenceTypeSalesOrder,
		ReferenceCode:     order.Code,
		ReferenceItemCode: &linha,
		ReservationDate:   time.Now(),
		Status:            "ACTIVE",
		CreatedBy:         order.CreatedBy,
	}
	if _, err := estoque.CreateReservation(ctx, res); err != nil {
		// O depósito do pedido não aceita reservas: o pedido segue sem reservar.
		if errors.Is(err, stockrepo.ErrAlmoxSemReserva) {
			return nil
		}
		return fmt.Errorf("reservando o item %d: %w", it.ItemCode, err)
	}
	return nil
}
