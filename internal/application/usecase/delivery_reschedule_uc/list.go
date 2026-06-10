package delivery_reschedule_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
)

type ListDeliveryReschedulesUseCase struct {
	Repo repository.DeliveryRescheduleRepository
	Auth ports.AuthService
}

func (uc *ListDeliveryReschedulesUseCase) Execute(
	ctx context.Context,
	orderCode int64) ([]*response.DeliveryRescheduleResponse, error) {
	if !uc.Auth.CanListDeliveryReschedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListByOrder(ctx, orderCode)
	if err != nil {
		return nil, err
	}
	return toDeliveryRescheduleResponses(list), nil
}
