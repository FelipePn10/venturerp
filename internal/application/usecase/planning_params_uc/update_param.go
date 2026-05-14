package planning_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
)

type UpdatePlanningParamUseCase struct {
	Repo repository.PlanningParamRepository
	Auth ports.AuthService
}

func (uc *UpdatePlanningParamUseCase) Execute(
	ctx context.Context,
	dto request.UpdatePlanningParamDTO,
) (*entity.PlanningParam, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.Update(ctx, dto.ParamNumber, dto.Value, dto.UpdatedBy)
}
