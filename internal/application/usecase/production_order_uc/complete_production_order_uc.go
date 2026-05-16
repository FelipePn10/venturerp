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

type CompleteProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *CompleteProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CompleteProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	endDate, _ := time.Parse("2006-01-02", dto.EndDate)

	return uc.Repo.Complete(ctx, dto.ID, endDate)
}
