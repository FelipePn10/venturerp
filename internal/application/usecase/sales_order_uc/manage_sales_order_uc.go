package sales_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	demandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	demandrepo "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type CancelSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CancelSalesOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Cancel(ctx, code)
}

type BlockSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *BlockSalesOrderUseCase) Execute(ctx context.Context, dto request.BlockSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Block(ctx, dto.Code, dto.Reason)
}

type UnblockSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *UnblockSalesOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Unblock(ctx, code)
}

type ChangeStatusSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
	// DemandRepo is optional. When set, moving the order to "Pedido" (confirmed)
	// automatically feeds the MRP by creating an independent demand per order
	// line, so the planner no longer has to register demand by hand.
	DemandRepo demandrepo.IndependentDemandRepository
	// CreditChecker is optional. When set, confirming an order runs an automatic
	// credit-limit check; an order that exceeds the customer's limit is blocked
	// (and does not feed the MRP) instead of flowing through unchecked.
	CreditChecker *CreditChecker
	// Reserver is optional. When set, confirming an order reserves available
	// stock per line (ATP) so the promise is backed by real availability.
	Reserver *OrderStockReserver
}

func (uc *ChangeStatusSalesOrderUseCase) Execute(ctx context.Context, dto request.ChangeStatusDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	newStatus := entity.SalesOrderStatus(dto.Status)
	if err := uc.Repo.ChangeStatus(ctx, dto.Code, newStatus); err != nil {
		return err
	}

	if newStatus != entity.SalesOrderStatusOrder {
		return nil
	}

	// On confirmation: run the credit check first. A blocked order must not feed
	// the MRP nor reserve stock.
	approved := true
	if uc.CreditChecker != nil {
		approved = uc.CreditChecker.Check(ctx, dto.Code)
	}
	if !approved {
		return nil
	}

	// Project each open order line as MRP demand and reserve available stock.
	if uc.DemandRepo != nil {
		uc.generateDemands(ctx, dto.Code)
	}
	if uc.Reserver != nil {
		uc.Reserver.Reserve(ctx, dto.Code)
	}
	return nil
}

// generateDemands creates one independent demand per open order line. It is
// best-effort: failures (including re-confirmation duplicates, since the demand
// code is derived deterministically from the order line) are ignored so they
// never block the status change itself.
func (uc *ChangeStatusSalesOrderUseCase) generateDemands(ctx context.Context, code int64) {
	order, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return
	}
	items, err := uc.Repo.ListItems(ctx, code)
	if err != nil {
		return
	}
	for _, it := range items {
		if !it.IsActive || it.Status == entity.SalesOrderItemStatusCancelled {
			continue
		}
		qty := it.RequestedQty
		if qty <= 0 {
			continue
		}

		demandDate := time.Now()
		switch {
		case it.DeliveryDate != nil:
			demandDate = *it.DeliveryDate
		case order.DeliveryDate != nil:
			demandDate = *order.DeliveryDate
		}

		var mask *string
		if it.Mask != "" {
			m := it.Mask
			mask = &m
		}

		// Deterministic, order-line-scoped code keeps re-confirmation idempotent.
		demandCode := code*100000 + int64(it.Sequence)
		demand := &demandentity.IndependentDemand{
			CodeDemand: demandCode,
			ItemCode:   it.ItemCode,
			Mask:       mask,
			Quantity:   qty,
			DemandDate: demandDate,
			CreatedBy:  order.CreatedBy,
		}
		_, _ = uc.DemandRepo.Create(ctx, demand)
	}
}
