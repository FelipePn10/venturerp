package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type GetProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetProductionOrderUseCase) Execute(ctx context.Context, id int64) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetByCode(ctx, id)
}

type GetAppointmentsUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetAppointmentsUseCase) Execute(ctx context.Context, productionOrderID int64) ([]*entity.ProductionAppointment, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetAppointments(ctx, productionOrderID)
}

type GetConsumptionsUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetConsumptionsUseCase) Execute(ctx context.Context, productionOrderID int64) ([]*entity.ProductionConsumption, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetConsumptions(ctx, productionOrderID)
}
