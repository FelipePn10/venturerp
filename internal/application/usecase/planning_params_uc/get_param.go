package planning_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
)

type GetPlanningParamUseCase struct {
	Repo repository.PlanningParamRepository
	Auth ports.AuthService
}

func (uc *GetPlanningParamUseCase) Execute(ctx context.Context, paramNumber int) (*response.PlanningParamResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.GetByNumber(ctx, paramNumber)
	if err != nil {
		return nil, err
	}
	return toPlanningParamResponse(p), nil
}
