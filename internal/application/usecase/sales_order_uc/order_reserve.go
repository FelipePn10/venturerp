package sales_order_uc

import (
	"context"
	"time"

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

// Reserve is best-effort: failures on individual lines never block the order
// confirmation flow.
func (r *OrderStockReserver) Reserve(ctx context.Context, code int64) {
	if already, err := r.StockRepo.HasActiveReservationByReference(ctx, stockentity.ReferenceTypeSalesOrder, code); err == nil && already {
		return
	}

	order, err := r.SalesRepo.GetByCode(ctx, code)
	if err != nil {
		return
	}
	items, err := r.SalesRepo.ListItems(ctx, code)
	if err != nil {
		return
	}

	for _, it := range items {
		if !it.IsActive || it.Status == entity.SalesOrderItemStatusCancelled || it.WarehouseCode == nil {
			continue
		}
		wanted := it.RequestedQty - it.AttendedQty - it.CancelledQty
		if wanted <= 0 {
			continue
		}

		available := 0.0
		if bal, balErr := r.StockRepo.GetBalance(ctx, it.ItemCode, it.Mask, *it.WarehouseCode); balErr == nil {
			available = bal.AvailableQty
		}
		reserveQty := wanted
		if available < reserveQty {
			reserveQty = available
		}
		if reserveQty <= 0 {
			continue
		}

		itemCode := it.ItemCode
		res := &stockentity.StockReservation{
			ItemCode:          it.ItemCode,
			Mask:              it.Mask,
			WarehouseID:       *it.WarehouseCode,
			Quantity:          reserveQty,
			ReferenceType:     stockentity.ReferenceTypeSalesOrder,
			ReferenceCode:     code,
			ReferenceItemCode: &itemCode,
			ReservationDate:   time.Now(),
			Status:            "ACTIVE",
			CreatedBy:         order.CreatedBy,
		}
		_, _ = r.StockRepo.CreateReservation(ctx, res)
	}
}
