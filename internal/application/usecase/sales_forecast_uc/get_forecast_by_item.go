package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type GetForecastByItemUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *GetForecastByItemUseCase) Execute(
	ctx context.Context,
	itemCode int64,
) ([]*response.SalesForecastResponse, error) {
	if !uc.Auth.CanListSalesForecasts(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	forecasts, err := uc.Repo.GetForecastByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toSalesForecastResponses(forecasts), nil
}
