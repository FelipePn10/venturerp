package sales_forecast_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type CreateForecastBlockUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *CreateForecastBlockUseCase) Execute(
	ctx context.Context,
	dto request.CreateForecastBlockDTO,
) (*response.SalesForecastBlockResponse, error) {
	if !uc.Auth.CanCreateForecastBlock(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	startDate, err := dataDoBloqueio(dto.StartDate, "data inicial do bloqueio")
	if err != nil {
		return nil, err
	}
	endDate, err := dataDoBloqueio(dto.EndDate, "data final do bloqueio")
	if err != nil {
		return nil, err
	}

	block, err := entity.NewSalesForecastBlock(startDate, endDate, dto.Reason, userID)
	if err != nil {
		return nil, erroDeRegra(err)
	}

	created, err := uc.Repo.CreateBlock(ctx, block)
	if err != nil {
		return nil, erroDeRegra(err)
	}
	return toForecastBlockResponse(created), nil
}
