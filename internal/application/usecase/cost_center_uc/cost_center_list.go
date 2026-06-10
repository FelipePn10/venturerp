package cost_center_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
)

type ListCostCentersUseCase struct {
	Repo repository.CostCenterRepository
	Auth ports.AuthService
}

func (uc *ListCostCentersUseCase) Execute(
	ctx context.Context,
) ([]*response.CostCenterResponse, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toCostCenterResponses(list), nil
}
