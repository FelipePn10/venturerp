package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type StartProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *StartProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.StartProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	startDate, _ := time.Parse("2006-01-02", dto.StartDate)

	return uc.Repo.Start(ctx, dto.ID, startDate)
}
