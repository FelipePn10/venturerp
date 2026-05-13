package planned_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
)

type FirmPlannedOrderUseCase struct {
	Repo repository.PlannedOrderRepository
	Auth ports.AuthService
}

func (uc *FirmPlannedOrderUseCase) Execute(ctx context.Context, dto request.FirmOrderDTO) (*entity.PlannedOrder, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.FirmOrder(ctx, dto.OrderCode)
}
