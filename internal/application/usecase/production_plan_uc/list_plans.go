package production_plan_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

type ListProductionPlansUseCase struct {
	Repo repository.ProductionPlanRepository
	Auth ports.AuthService
}

func (uc *ListProductionPlansUseCase) Execute(ctx context.Context) ([]*entity.ProductionPlan, error) {
	if !uc.Auth.CanListProductionPlans(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
