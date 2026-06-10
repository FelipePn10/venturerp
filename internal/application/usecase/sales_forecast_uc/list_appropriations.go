package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type ListAppropriationTablesUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *ListAppropriationTablesUseCase) Execute(
	ctx context.Context,
) ([]*response.AppropriationTableResponse, error) {
	if !uc.Auth.CanListAppropriationTables(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	tables, err := uc.Repo.ListAppropriations(ctx)
	if err != nil {
		return nil, err
	}
	return toAppropriationTableResponses(tables), nil
}
