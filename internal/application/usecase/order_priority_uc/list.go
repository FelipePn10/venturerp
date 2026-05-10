package order_priority_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository"
)

type ListOrderPrioritiesUseCase struct {
	Repo repository.OrderPriorityRepository
	Auth ports.AuthService
}

func (uc *ListOrderPrioritiesUseCase) Execute(
	ctx context.Context,
) ([]*entity.OrderPriority, error) {
	if !uc.Auth.CanOrderPriority(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
