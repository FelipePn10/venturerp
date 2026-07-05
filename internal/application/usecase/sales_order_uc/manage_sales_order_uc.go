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
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CancelSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CancelSalesOrderUseCase) Execute(ctx context.Context, dto request.CancelSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.Cancel(ctx, dto.Code, dto.Reason, dto.Complement)
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

type AnalyzeSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *AnalyzeSalesOrderUseCase) Execute(ctx context.Context, dto request.AnalyzeSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	area := dto.Area
	if area != "COMMERCIAL" && area != "FINANCIAL" {
		return errorsuc.NewValidationError("area must be COMMERCIAL or FINANCIAL")
	}
	status := entity.SalesOrderAnalysisStatus(dto.Status)
	if status != entity.SalesOrderAnalysisApproved && status != entity.SalesOrderAnalysisRejected && status != entity.SalesOrderAnalysisNotAnalyzed {
		return errorsuc.NewValidationError("invalid analysis status")
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.Analyze(ctx, dto.Code, area, status, dto.Reason, dto.CreatedBy)
}

type ReleaseSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ReleaseSalesOrderUseCase) Execute(ctx context.Context, dto request.ReleaseSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesOrderReleaseStatus(dto.ReleaseStatus)
	if status != entity.SalesOrderReleaseBlocked && status != entity.SalesOrderReleaseManual && status != entity.SalesOrderReleaseOK {
		return errorsuc.NewValidationError("invalid release_status")
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.Release(ctx, dto.Code, status, dto.Reason, dto.Area, dto.CreatedBy)
}

type AttendSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *AttendSalesOrderUseCase) Execute(ctx context.Context, dto request.AttendSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	eventDate := datetime.ParseDatePtr(&dto.EventDate)
	return uc.Repo.Attend(ctx, dto.Code, dto.Reason, eventDate, dto.CreatedBy)
}

type ConferSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ConferSalesOrderUseCase) Execute(ctx context.Context, dto request.ConferSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesOrderConferenceStatus(dto.Status)
	if status != entity.SalesOrderConferencePending && status != entity.SalesOrderConferenceConferred && status != entity.SalesOrderConferenceDivergent {
		return errorsuc.NewValidationError("invalid conference status")
	}
	return uc.Repo.Confer(ctx, dto.Code, status, dto.Reason, dto.CreatedBy)
}

type SaveSalesOrderDelayReasonUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *SaveSalesOrderDelayReasonUseCase) Execute(ctx context.Context, dto request.SaveSalesOrderDelayReasonDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.Reason == "" || dto.Action == "" {
		return errorsuc.NewValidationError("reason and action are required")
	}
	return uc.Repo.SaveDelayReason(ctx, dto.Code, dto.Reason, dto.Action, dto.CreatedBy)
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
