package delivery_promise_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/repository"
)

type ManageDeliveryPromiseParamsUseCase struct {
	Repo repository.DeliveryPromiseParamsRepository
	Auth ports.AuthService
}

func (uc *ManageDeliveryPromiseParamsUseCase) Get(ctx context.Context) (*response.DeliveryPromiseParamsResponse, error) {
	p, err := uc.Repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	return toDeliveryPromiseParamsResponse(p), nil
}

func (uc *ManageDeliveryPromiseParamsUseCase) Save(ctx context.Context, dto request.UpdateDeliveryPromiseParamsDTO, userID string) (*response.DeliveryPromiseParamsResponse, error) {
	params := &entity.DeliveryPromiseParams{
		UseDeliveryPromise:      dto.UseDeliveryPromise,
		BlockedOrdersInPromise:  dto.BlockedOrdersInPromise,
		DefaultOrderSort:        dto.DefaultOrderSort,
		ShowOrderValues:         dto.ShowOrderValues,
		BlockedExportInPromise:  dto.BlockedExportInPromise,
		BreakTankOccupation:     dto.BreakTankOccupation,
		RecalculateAfterRelease: dto.RecalculateAfterRelease,
		ReprogramLoadedOrders:   dto.ReprogramLoadedOrders,
		AllowDeliveryDateChange: dto.AllowDeliveryDateChange,
		UpdatedBy:               dto.UpdatedBy,
	}
	saved, err := uc.Repo.Save(ctx, params)
	if err != nil {
		return nil, err
	}
	return toDeliveryPromiseParamsResponse(saved), nil
}
