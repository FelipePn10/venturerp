package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
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
	if dto.ID == 0 {
		return nil, errorsuc.NewValidationError("id is required")
	}

	// Default to the real start moment (today) when no valid date is sent,
	// instead of persisting the zero time (0001-01-01).
	startDate := datetime.ParseDateOrDefault(dto.StartDate, time.Now())

	return uc.Repo.Start(ctx, dto.ID, startDate)
}
