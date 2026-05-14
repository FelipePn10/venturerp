package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type CreateAppropriationTableUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *CreateAppropriationTableUseCase) Execute(
	ctx context.Context,
	dto request.CreateAppropriationTableDTO,
) (*entity.AppropriationTable, error) {
	if !uc.Auth.CanCreateAppropriationTable(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	table, err := entity.NewAppropriationTable(
		dto.Description,
		dto.MondayPct,
		dto.TuesdayPct,
		dto.WednesdayPct,
		dto.ThursdayPct,
		dto.FridayPct,
		dto.SaturdayPct,
		dto.SundayPct,
		dto.IsDefault,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return uc.Repo.CreateAppropriation(ctx, table)
}
