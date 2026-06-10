package planned_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type FirmPlannedOrderUseCase struct {
	Repo repository.PlannedOrderRepository
	Auth ports.AuthService
	// ProdOrderRepo is optional. When set, firming a PRODUCTION planned order
	// also creates the corresponding Production Order (OF), mirroring the
	// approve→purchase-order flow already in place on the purchasing side.
	ProdOrderRepo productionrepo.ProductionOrderRepository
}

func (uc *FirmPlannedOrderUseCase) Execute(ctx context.Context, dto request.FirmOrderDTO) (*response.PlannedOrderResponse, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// Check the prior state so the OF is generated only on the first firming,
	// avoiding duplicate production orders if the endpoint is called again.
	wasFirm := false
	if uc.ProdOrderRepo != nil {
		if prev, err := uc.Repo.GetByCode(ctx, dto.OrderCode); err == nil {
			wasFirm = prev.IsFirm
		}
	}

	order, err := uc.Repo.FirmOrder(ctx, dto.OrderCode)
	if err != nil {
		return nil, err
	}

	if uc.ProdOrderRepo != nil && !wasFirm && order.OrderType == types.OrderProduction {
		if ofErr := uc.createProductionOrder(ctx, order); ofErr != nil {
			return nil, ofErr
		}
	}

	return toPlannedOrderResponse(order), nil
}

// createProductionOrder builds the OF from the firmed planned order, mirroring
// the manual CreateProductionOrderUseCase. It links back to the planned order
// via its code so the production side stays traceable to the plan.
func (uc *FirmPlannedOrderUseCase) createProductionOrder(ctx context.Context, order *entity.PlannedOrder) error {
	nextNum, err := uc.ProdOrderRepo.GetNextOrderNumber(ctx)
	if err != nil {
		nextNum = 1
	}

	mask := ""
	if order.Mask != nil {
		mask = *order.Mask
	}
	plannedCode := order.Code

	of := &productionentity.ProductionOrder{
		OrderNumber:    nextNum,
		PlannedOrderID: &plannedCode,
		ItemCode:       order.ItemCode,
		Mask:           mask,
		PlannedQty:     order.Quantity,
		Status:         productionentity.StatusOpen,
		CostCenterID:   order.CostCenterCode,
		EmployeeID:     order.EmployeeCode,
		MachineID:      order.MachineCode,
		Priority:       order.Priority,
		Notes:          order.Notes,
		StartDate:      order.StartDate,
		EndDate:        order.EndDate,
		CreatedBy:      order.CreatedBy,
	}
	_, err = uc.ProdOrderRepo.Create(ctx, of)
	return err
}
