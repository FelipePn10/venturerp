package production_plan_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

type GetProductionPlanUseCase struct {
	Repo repository.ProductionPlanRepository
	Auth ports.AuthService
}

func (uc *GetProductionPlanUseCase) Execute(ctx context.Context, code int64) (*response.ProductionPlanResponse, error) {
	if !uc.Auth.CanListProductionPlans(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toProductionPlanResponse(p), nil
}
