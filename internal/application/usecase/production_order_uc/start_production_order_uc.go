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
type productionOperationsChecker interface {
	HasProductionOperations(context.Context, int64) (bool, error)
}

func (uc *StartProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.StartProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ID == 0 {
		return nil, errorsuc.NewValidationError("informe o identificador")
	}
	if checker, ok := uc.Repo.(productionOperationsChecker); ok {
		hasOperations, err := checker.HasProductionOperations(ctx, dto.ID)
		if err != nil {
			return nil, err
		}
		if !hasOperations {
			return nil, errorsuc.NewValidationError("a ordem de produção não possui operações; gere as operações a partir de um roteiro aprovado antes de iniciar")
		}
	}

	// Default to the real start moment (today) when no valid date is sent,
	// instead of persisting the zero time (0001-01-01).
	startDate := datetime.ParseDateOrDefault(dto.StartDate, time.Now())

	return uc.Repo.Start(ctx, dto.ID, startDate)
}
