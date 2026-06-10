package order_priority_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository"
)

type FindPriorityByValueUseCase struct {
	Repo repository.OrderPriorityRepository
	Auth ports.AuthService
}

func (uc *FindPriorityByValueUseCase) Execute(
	ctx context.Context, value float64,
) (*response.OrderPriorityResponse, error) {
	if !uc.Auth.CanOrderPriority(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.FindByValue(ctx, value)
	if err != nil {
		return nil, err
	}
	return toOrderPriorityResponse(p), nil
}
