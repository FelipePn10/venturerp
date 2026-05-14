package sales_forecast_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type CreateSalesForecastUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *CreateSalesForecastUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesForecastDTO,
) (*entity.SalesForecast, error) {
	if !uc.Auth.CanCreateSalesForecast(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	// Determine representative date for the given week/year to check blocks.
	// Use the first day of the week (approximate: Jan 1 + (week-1)*7 days).
	checkDate, err := weekToDate(dto.Year, dto.Week)
	if err != nil {
		return nil, fmt.Errorf("invalid week/year combination: %w", err)
	}

	blocked, err := uc.Repo.IsBlocked(ctx, checkDate)
	if err != nil {
		return nil, fmt.Errorf("checking forecast period: %w", err)
	}
	if blocked {
		return nil, fmt.Errorf("forecast period week %d of year %d is blocked", dto.Week, dto.Year)
	}

	forecast, err := entity.NewSalesForecast(
		dto.ItemCode,
		dto.Mask,
		dto.Week,
		dto.Year,
		dto.Quantity,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return uc.Repo.CreateForecast(ctx, forecast)
}
