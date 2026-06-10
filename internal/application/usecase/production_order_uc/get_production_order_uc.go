package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type GetProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetProductionOrderUseCase) Execute(ctx context.Context, id int64) (*response.ProductionOrderResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	o, err := uc.Repo.GetByCode(ctx, id)
	if err != nil {
		return nil, err
	}
	return toProductionOrderResponse(o), nil
}

type GetAppointmentsUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetAppointmentsUseCase) Execute(ctx context.Context, productionOrderID int64) ([]*response.ProductionAppointmentResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.GetAppointments(ctx, productionOrderID)
	if err != nil {
		return nil, err
	}
	return toProductionAppointmentResponses(list), nil
}

type GetConsumptionsUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetConsumptionsUseCase) Execute(ctx context.Context, productionOrderID int64) ([]*response.ProductionConsumptionResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.GetConsumptions(ctx, productionOrderID)
	if err != nil {
		return nil, err
	}
	return toProductionConsumptionResponses(list), nil
}
