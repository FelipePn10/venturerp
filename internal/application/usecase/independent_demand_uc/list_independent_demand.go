package independent_demand_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
)

type ListIndependentDemandsUseCase struct {
	Repo repository.IndependentDemandRepository
	Auth ports.AuthService
}

func (uc *ListIndependentDemandsUseCase) Execute(
	ctx context.Context,
) ([]*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanListIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toIndependentDemandResponses(list), nil
}
