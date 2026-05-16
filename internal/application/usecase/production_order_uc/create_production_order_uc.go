package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
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

	if dto.StartDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.StartDate)
		order.StartDate = &t
	}
	if dto.EndDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.EndDate)
		order.EndDate = &t
	}

	return uc.Repo.Create(ctx, order)
}
