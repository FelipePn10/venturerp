package planned_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
)

type CreatePlannedOrderUseCase struct {
	Repo repository.PlannedOrderRepository
	Auth ports.AuthService
}

func (uc *CreatePlannedOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreatePlannedOrderDTO,
) (*entity.PlannedOrder, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	needDate, _ := time.Parse("2006-01-02", dto.NeedDate)

	nextNum, err := uc.Repo.GetNextOrderNumber(ctx)
	if err != nil {
		nextNum = 1
	}

	order := &entity.PlannedOrder{
		OrderNumber:    nextNum,
		ItemCode:       dto.ItemCode,
		Mask:           dto.Mask,
		Quantity:       dto.Quantity,
		OrderType:      types.OrderType(dto.OrderType),
		Status:         types.StatusPlanned,
		NeedDate:       needDate,
		CostCenterCode: dto.CostCenterCode,
		EmployeeCode:   dto.EmployeeCode,
		MachineCode:    dto.MachineCode,
		ProductionTime: dto.ProductionTime,
		Notes:          dto.Notes,
		CreatedBy:      dto.CreatedBy,
	}
	return uc.Repo.Create(ctx, order)
}
