package planning_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
)

type ListPlanningParamsUseCase struct {
	Repo repository.PlanningParamRepository
	Auth ports.AuthService
}

func (uc *ListPlanningParamsUseCase) Execute(ctx context.Context) ([]*entity.PlanningParam, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
