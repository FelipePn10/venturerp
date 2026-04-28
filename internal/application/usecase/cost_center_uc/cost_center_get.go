package cost_center_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
)

type GetCostCenterUseCase struct {
	Repo repository.CostCenterRepository
	Auth ports.AuthService
}

func (uc *GetCostCenterUseCase) Execute(
	ctx context.Context,
	costCenterCode int32,
) (*entity.CostCenter, error) {
	if !uc.Auth.CanGetCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetByCode(ctx, costCenterCode)
}
