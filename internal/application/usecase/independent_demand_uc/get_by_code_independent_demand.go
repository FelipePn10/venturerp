package independent_demand_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
)

type GetIndependentDemandByCodeUseCase struct {
	Repo repository.IndependentDemandRepository
	Auth ports.AuthService
}

func (uc *GetIndependentDemandByCodeUseCase) Execute(
	ctx context.Context,
	code int64,
) (*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanViewIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	d, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toIndependentDemandResponse(d), nil
}
