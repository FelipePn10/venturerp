package delivery_reschedule_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
)

type CreateDeliveryRescheduleUseCase struct {
	Repo repository.DeliveryRescheduleRepository
	Auth ports.AuthService
}

func (uc *CreateDeliveryRescheduleUseCase) Execute(
	ctx context.Context,
	dto request.CreateDeliveryRescheduleDTO,
) (*response.DeliveryRescheduleResponse, error) {
	if !uc.Auth.CanCreateDeliveryReschedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	r := &entity.DeliveryReschedule{
		Code:           dto.Code,
		SalesOrderCode: dto.SalesOrderCode,
		ItemCode:       dto.ItemCode,
		OldDate:        dto.OldDate,
		NewDate:        dto.NewDate,
		Reason:         dto.Reason,
		CreatedBy:      dto.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, r)
	if err != nil {
		return nil, err
	}
	return toDeliveryRescheduleResponse(created), nil
}
