package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type CloseProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// SettleUC is optional. When set, closing the order also settles its actual
	// cost (material + conversion) and variance against the standard, so a closed
	// OF always carries its real cost without a manual step.
	SettleUC *SettleProductionCostUseCase
}

func (uc *CloseProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CloseProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	order, err := uc.Repo.Close(ctx, dto.ID)
	if err != nil {
		return nil, err
	}

	// Settle the actual cost on close. Best-effort: a costing failure must not
	// undo the close, which is the authoritative state transition.
	if uc.SettleUC != nil {
		_, _ = uc.SettleUC.Execute(ctx, dto.ID)
	}

	return order, nil
}
