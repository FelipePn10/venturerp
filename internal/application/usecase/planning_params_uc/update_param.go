package planning_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
)

type UpdatePlanningParamUseCase struct {
	Repo repository.PlanningParamRepository
	Auth ports.AuthService
}

func (uc *UpdatePlanningParamUseCase) Execute(
	ctx context.Context,
	dto request.UpdatePlanningParamDTO,
) (*response.PlanningParamResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.Update(ctx, dto.ParamNumber, dto.Value, dto.UpdatedBy)
	if err != nil {
		return nil, err
	}
	return toPlanningParamResponse(p), nil
}
