package planned_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
)

type ListPlannedOrdersUseCase struct {
	Repo repository.PlannedOrderRepository
	Auth ports.AuthService
}

func (uc *ListPlannedOrdersUseCase) Execute(ctx context.Context) ([]*entity.PlannedOrder, error) {
	if !uc.Auth.CanListOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
