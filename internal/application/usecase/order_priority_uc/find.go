package order_priority_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository"
)

type FindPriorityByValueUseCase struct {
	Repo repository.OrderPriorityRepository
	Auth ports.AuthService
}

func (uc *FindPriorityByValueUseCase) Execute(
	ctx context.Context, value float64,
) (*entity.OrderPriority, error) {
	if !uc.Auth.CanOrderPriority(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.FindByValue(ctx, value)
}
