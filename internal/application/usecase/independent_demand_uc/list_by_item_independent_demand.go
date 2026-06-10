package independent_demand_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
)

type ListIndependentDemandByItemUseCase struct {
	Repo repository.IndependentDemandRepository
	Auth ports.AuthService
}

func (uc *ListIndependentDemandByItemUseCase) Execute(
	ctx context.Context,
	itemCode int64,
) ([]*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanViewIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.ListByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toIndependentDemandResponses(list), nil
}
