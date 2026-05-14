package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type SetDefaultAppropriationUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *SetDefaultAppropriationUseCase) Execute(
	ctx context.Context,
	id int64,
) error {
	if !uc.Auth.CanCreateAppropriationTable(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.SetDefaultAppropriation(ctx, id)
}
