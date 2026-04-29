package delivery_promise_params_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/repository"
)

type ManageDeliveryPromiseParamsUseCase struct {
	Repo repository.DeliveryPromiseParamsRepository
	Auth ports.AuthService
}

func (uc *ManageDeliveryPromiseParamsUseCase) Get(ctx context.Context) (*entity.DeliveryPromiseParams, error) {
	return uc.Repo.Get(ctx)
}

func (uc *ManageDeliveryPromiseParamsUseCase) Save(ctx context.Context, dto request.UpdateDeliveryPromiseParamsDTO, userID string) (*entity.DeliveryPromiseParams, error) {
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
	return uc.Repo.Save(ctx, params)
}
