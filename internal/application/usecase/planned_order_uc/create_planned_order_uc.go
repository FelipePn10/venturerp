package planned_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
) (*response.PlannedOrderResponse, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode == 0 {
		return nil, errorsuc.NewValidationError("item_code is required")
	}
	if dto.Quantity <= 0 {
		return nil, errorsuc.NewValidationError("quantity must be greater than zero")
	}

	// demand_type_enum is NOT NULL with no default; a manual planned order is
	// independent demand unless the caller states otherwise. Validate the value
	// so an unknown string returns 422 instead of the raw enum error.
	demandType := types.DemandIndependent
	if dto.DemandType != "" {
		demandType = types.DemandType(dto.DemandType)
		switch demandType {
		case types.DemandSalesOrder, types.DemandForecast, types.DemandIndependent,
			types.DemandSafetyStock, types.DemandReplenishment:
		default:
			return nil, errorsuc.NewValidationError("invalid demand_type: must be SALES_ORDER, FORECAST, INDEPENDENT, SAFETY_STOCK or REPLENISHMENT")
		}
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
		DemandType:     demandType,
		Status:         types.StatusPlanned,
		NeedDate:       needDate,
		CostCenterCode: dto.CostCenterCode,
		EmployeeCode:   dto.EmployeeCode,
		MachineCode:    dto.MachineCode,
		ProductionTime: dto.ProductionTime,
		Notes:          dto.Notes,
		CreatedBy:      dto.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, order)
	if err != nil {
		return nil, err
	}
	return toPlannedOrderResponse(created), nil
}
