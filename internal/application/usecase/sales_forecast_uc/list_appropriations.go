package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type ListAppropriationTablesUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *ListAppropriationTablesUseCase) Execute(
	ctx context.Context,
) ([]*entity.AppropriationTable, error) {
	if !uc.Auth.CanListAppropriationTables(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListAppropriations(ctx)
}
