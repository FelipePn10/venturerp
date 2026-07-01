package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CreateProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *CreateProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreateProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode == 0 {
		return nil, errorsuc.NewValidationError("item_code is required")
	}
	if dto.PlannedQty <= 0 {
		return nil, errorsuc.NewValidationError("planned_qty must be greater than zero")
	}

	nextNum, err := uc.Repo.GetNextOrderNumber(ctx)
	if err != nil {
		nextNum = 1
	}

	order := &entity.ProductionOrder{
		OrderNumber:    nextNum,
		PlannedOrderID: dto.PlannedOrderID,
		ItemCode:       dto.ItemCode,
		Mask:           dto.Mask,
		PlannedQty:     dto.PlannedQty,
		Status:         entity.StatusOpen,
		MachineID:      dto.MachineID,
		CostCenterID:   dto.CostCenterID,
		EmployeeID:     dto.EmployeeID,
		Priority:       dto.Priority,
		Notes:          dto.Notes,
		CreatedBy:      dto.CreatedBy,
	}

	order.StartDate = datetime.ParseDatePtr(dto.StartDate)
	order.EndDate = datetime.ParseDatePtr(dto.EndDate)

	return uc.Repo.Create(ctx, order)
}
