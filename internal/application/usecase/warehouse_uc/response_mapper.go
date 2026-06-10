package warehouse_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity"
)

func toWarehouseResponse(w *entity.Warehouse) *response.WarehouseResponse {
	if w == nil {
		return nil
	}
	return &response.WarehouseResponse{
		ID:                  w.ID,
		Code:                w.Code,
		Description:         w.Description,
		Location:            w.Location.String(),
		Type:                w.Type.String(),
		Disposition:         w.Disposition,
		ReservationsAllowed: w.ReservationsAllowed,
		CreatedBy:           w.CreatedBy,
		CreatedAt:           w.CreatedAt,
	}
}
