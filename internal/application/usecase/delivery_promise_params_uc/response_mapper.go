package delivery_promise_params_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
)

func toDeliveryPromiseParamsResponse(p *entity.DeliveryPromiseParams) *response.DeliveryPromiseParamsResponse {
	if p == nil {
		return nil
	}
	return &response.DeliveryPromiseParamsResponse{
		ID:                      p.ID,
		UseDeliveryPromise:      p.UseDeliveryPromise,
		BlockedOrdersInPromise:  p.BlockedOrdersInPromise,
		DefaultOrderSort:        p.DefaultOrderSort,
		ShowOrderValues:         p.ShowOrderValues,
		BlockedExportInPromise:  p.BlockedExportInPromise,
		BreakTankOccupation:     p.BreakTankOccupation,
		RecalculateAfterRelease: p.RecalculateAfterRelease,
		ReprogramLoadedOrders:   p.ReprogramLoadedOrders,
		AllowDeliveryDateChange: p.AllowDeliveryDateChange,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
		UpdatedBy:               p.UpdatedBy,
	}
}
