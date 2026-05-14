package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type GetForecastByItemUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *GetForecastByItemUseCase) Execute(
	ctx context.Context,
	itemCode int64,
) ([]*entity.SalesForecast, error) {
	if !uc.Auth.CanListSalesForecasts(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetForecastByItem(ctx, itemCode)
}
