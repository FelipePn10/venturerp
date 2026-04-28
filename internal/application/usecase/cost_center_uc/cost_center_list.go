package cost_center_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
)

type ListCostCentersUseCase struct {
	Repo repository.CostCenterRepository
	Auth ports.AuthService
}

func (uc *ListCostCentersUseCase) Execute(
	ctx context.Context,
) ([]*entity.CostCenter, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
