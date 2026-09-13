package sales_forecast_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
) (*response.SalesForecastResponse, error) {
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
		return nil, errorsuc.NewValidationError(fmt.Sprintf("semana e ano inválidos: %v", err))
	}

	blocked, err := uc.Repo.IsBlocked(ctx, checkDate)
	if err != nil {
		return nil, fmt.Errorf("verificando o período de previsão: %w", err)
	}
	if blocked {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a semana %d de %d está bloqueada para previsão", dto.Week, dto.Year))
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
		return nil, erroDeRegra(err)
	}

	created, err := uc.Repo.CreateForecast(ctx, forecast)
	if err != nil {
		return nil, erroDeRegra(err)
	}
	return toSalesForecastResponse(created), nil
}
