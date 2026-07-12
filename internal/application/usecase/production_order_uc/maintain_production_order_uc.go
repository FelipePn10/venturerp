package production_order_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type MaintainProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *MaintainProductionOrderUseCase) Execute(ctx context.Context, dto request.MaintainProductionOrderDTO) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if phase4, ok := uc.Repo.(interface {
		GetMaintenance(context.Context, *int64) ([]entity.ProductionOrderMaintenanceView, error)
	}); ok {
		views, err := phase4.GetMaintenance(ctx, &dto.ID)
		if err != nil {
			return nil, err
		}
		if len(views) != 1 {
			return nil, fmt.Errorf("production order is not available for maintenance")
		}
	}
	order, err := uc.Repo.GetByCode(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	if order.Status == entity.StatusCompleted || order.Status == entity.StatusClosed || order.Status == entity.StatusCancelled {
		return nil, fmt.Errorf("closed, completed or cancelled production order cannot be maintained")
	}
	activity, err := uc.Repo.HasProductionActivity(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	if activity {
		return nil, fmt.Errorf("production order with movement, appointment, consumption or WMS separation cannot be maintained")
	}
	if dto.PlannedQty != nil {
		allowed, err := uc.Repo.CanChangeOrderQuantity(ctx, dto.ID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, fmt.Errorf("production parameter 10 does not allow quantity changes")
		}
		if !dto.PlannedQty.IsPositive() || dto.PlannedQty.LessThan(decimal.NewFromFloat(order.ProducedQty)) {
			return nil, fmt.Errorf("planned quantity must be positive and not lower than produced quantity")
		}
		fractional, err := uc.Repo.AcceptsFractionalQuantity(ctx, order.ItemCode)
		if err != nil {
			return nil, err
		}
		if !fractional && !dto.PlannedQty.Equal(dto.PlannedQty.Truncate(0)) {
			return nil, fmt.Errorf("item does not accept fractional quantity")
		}
		order.PlannedQty, _ = dto.PlannedQty.Float64()
	}
	if dto.StartDate != nil || dto.EndDate != nil {
		allowed, err := uc.Repo.CanChangeOrderDates(ctx, dto.ID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, fmt.Errorf("production parameter 14 does not allow date changes")
		}
		if dto.StartDate != nil {
			order.StartDate = datetime.ParseDatePtr(dto.StartDate)
		}
		if dto.EndDate != nil {
			order.EndDate = datetime.ParseDatePtr(dto.EndDate)
		}
	}
	if dto.MachineID != nil {
		order.MachineID = dto.MachineID
	}
	if dto.Priority != nil {
		order.Priority = dto.Priority
	}
	if dto.Notes != nil {
		order.Notes = dto.Notes
	}
	return uc.Repo.Update(ctx, order)
}
