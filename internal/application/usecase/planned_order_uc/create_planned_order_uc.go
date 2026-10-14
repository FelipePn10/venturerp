package planned_order_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"

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
		return nil, errorsuc.NewValidationError("informe o item")
	}
	if dto.Quantity <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade deve ser maior que zero")
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
			return nil, errorsuc.NewValidationError("tipo de demanda inválido: use pedido de venda, previsão, independente, estoque de segurança ou reposição")
		}
	}

	needDate, msg := datetime.Obrigatoria("data de necessidade", dto.NeedDate)
	if msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}

	// Falha ao numerar é erro: o "1" de antes colidia com a ordem 1 existente.
	nextNum, err := uc.Repo.GetNextOrderNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("numerando a ordem planejada: %w", err)
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
