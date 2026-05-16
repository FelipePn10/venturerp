package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type ListProductionOrdersUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *ListProductionOrdersUseCase) Execute(ctx context.Context) ([]*entity.ProductionOrder, error) {
	if !uc.Auth.CanListOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.List(ctx)
}
