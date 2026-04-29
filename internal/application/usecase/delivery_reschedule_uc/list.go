package delivery_reschedule_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
)

type ListDeliveryReschedulesUseCase struct {
	Repo repository.DeliveryRescheduleRepository
	Auth ports.AuthService
}

func (uc *ListDeliveryReschedulesUseCase) Execute(
	ctx context.Context,
	orderCode int64) ([]*entity.DeliveryReschedule, error) {
	if !uc.Auth.CanListDeliveryReschedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListByOrder(ctx, orderCode)
}
