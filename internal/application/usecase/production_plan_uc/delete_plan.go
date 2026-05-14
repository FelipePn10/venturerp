package production_plan_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

type DeleteProductionPlanUseCase struct {
	Repo repository.ProductionPlanRepository
	Auth ports.AuthService
}

func (uc *DeleteProductionPlanUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanDeleteProductionPlan(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Delete(ctx, code)
}
