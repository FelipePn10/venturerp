package order_priority_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository"
)

type CreateOrderPriorityUseCase struct {
	Repo repository.OrderPriorityRepository
	Auth ports.AuthService
}

func (uc *CreateOrderPriorityUseCase) Execute(
	ctx context.Context,
	dto request.CreateOrderPriorityDTO,
) (*response.OrderPriorityResponse, error) {
	if !uc.Auth.CanCreateOrderPriority(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	op := &entity.OrderPriority{
		IntervalStart: dto.IntervalStart,
		IntervalEnd:   dto.IntervalEnd,
		Priority:      dto.Priority,
		Description:   dto.Description,
		CreatedBy:     dto.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, op)
	if err != nil {
		return nil, err
	}
	return toOrderPriorityResponse(created), nil
}
