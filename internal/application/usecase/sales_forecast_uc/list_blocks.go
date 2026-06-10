package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type ListForecastBlocksUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *ListForecastBlocksUseCase) Execute(
	ctx context.Context,
) ([]*response.SalesForecastBlockResponse, error) {
	if !uc.Auth.CanListForecastBlocks(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	blocks, err := uc.Repo.ListBlocks(ctx)
	if err != nil {
		return nil, err
	}
	return toForecastBlockResponses(blocks), nil
}
