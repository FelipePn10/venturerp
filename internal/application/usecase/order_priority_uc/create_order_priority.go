package order_priority_uc

import (
	"context"
	"fmt"
	"strings"

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
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.IntervalStart >= dto.IntervalEnd || strings.TrimSpace(dto.Priority) == "" {
		return nil, fmt.Errorf("interval_start must be lower than interval_end and priority is required")
	}
	existing, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, interval := range existing {
		if dto.IntervalStart <= interval.IntervalEnd && dto.IntervalEnd >= interval.IntervalStart {
			return nil, fmt.Errorf("priority interval overlaps or touches interval %d", interval.Code)
		}
	}
	op := &entity.OrderPriority{
		IntervalStart: dto.IntervalStart,
		IntervalEnd:   dto.IntervalEnd,
		Priority:      dto.Priority,
		Description:   dto.Description,
		CreatedBy:     userID,
	}
	created, err := uc.Repo.Create(ctx, op)
	if err != nil {
		return nil, err
	}
	return toOrderPriorityResponse(created), nil
}
