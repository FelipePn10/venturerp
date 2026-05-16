package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type AddConsumptionUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *AddConsumptionUseCase) Execute(
	ctx context.Context,
	dto request.AddConsumptionDTO,
) (*entity.ProductionConsumption, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	consumptionDate, _ := time.Parse("2006-01-02", dto.ConsumptionDate)

	consumption := &entity.ProductionConsumption{
		ProductionOrderID: dto.ProductionOrderID,
		AppointmentID:     dto.AppointmentID,
		ItemCode:          dto.ItemCode,
		ConsumedQty:       dto.ConsumedQty,
		WarehouseID:       dto.WarehouseID,
		Lot:               dto.Lot,
		ConsumptionDate:   consumptionDate,
		Notes:             dto.Notes,
		CreatedBy:         dto.CreatedBy,
	}

	return uc.Repo.AddConsumption(ctx, consumption)
}
