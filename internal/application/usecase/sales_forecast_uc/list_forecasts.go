package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type ListSalesForecastsUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *ListSalesForecastsUseCase) Execute(
	ctx context.Context,
	year int,
) ([]*response.SalesForecastResponse, error) {
	if !uc.Auth.CanListSalesForecasts(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	forecasts, err := uc.Repo.ListForecasts(ctx, year)
	if err != nil {
		return nil, err
	}
	return toSalesForecastResponses(forecasts), nil
}
