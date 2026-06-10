package independent_demand_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
)

type ListIndependentDemandFromDateUseCase struct {
	Repo repository.IndependentDemandRepository
	Auth ports.AuthService
}

func (uc *ListIndependentDemandFromDateUseCase) Execute(
	ctx context.Context,
	date time.Time,
) ([]*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanViewIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.ListFromDate(ctx, date)
	if err != nil {
		return nil, err
	}
	return toIndependentDemandResponses(list), nil
}
