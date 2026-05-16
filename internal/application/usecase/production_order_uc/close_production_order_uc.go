package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type CloseProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *CloseProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CloseProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.Close(ctx, dto.ID)
}
