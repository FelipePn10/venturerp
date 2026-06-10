package order_priority_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository"
)

type ListOrderPrioritiesUseCase struct {
	Repo repository.OrderPriorityRepository
	Auth ports.AuthService
}

func (uc *ListOrderPrioritiesUseCase) Execute(
	ctx context.Context,
) ([]*response.OrderPriorityResponse, error) {
	if !uc.Auth.CanOrderPriority(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toOrderPriorityResponses(list), nil
}
