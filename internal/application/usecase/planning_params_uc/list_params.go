package planning_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
)

type ListPlanningParamsUseCase struct {
	Repo repository.PlanningParamRepository
	Auth ports.AuthService
}

func (uc *ListPlanningParamsUseCase) Execute(ctx context.Context) ([]*response.PlanningParamResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toPlanningParamResponses(list), nil
}
