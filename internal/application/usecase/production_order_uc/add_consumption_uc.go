package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type AddConsumptionUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StockRepo is optional. When set, registering a consumption also posts an
	// OUT stock movement (which updates the balance) for the consumed item.
	StockRepo stockrepo.StockRepository
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

	saved, err := uc.Repo.AddConsumption(ctx, consumption)
	if err != nil {
		return nil, err
	}

	// Post the OUT movement for the consumed input so the warehouse balance is
	// reduced automatically. Requires a target warehouse on the consumption.
	if uc.StockRepo != nil && saved.WarehouseID != nil {
		refType := stockentity.ReferenceTypeProductionOrder
		refCode := saved.ProductionOrderID
		mov := &stockentity.StockMovement{
			ItemCode:      saved.ItemCode,
			WarehouseID:   *saved.WarehouseID,
			MovementType:  stockentity.MovementTypeOut,
			Quantity:      saved.ConsumedQty,
			ReferenceType: &refType,
			ReferenceCode: &refCode,
			Lot:           saved.Lot,
			Notes:         saved.Notes,
			CreatedBy:     saved.CreatedBy,
		}
		if _, moveErr := uc.StockRepo.CreateMovement(ctx, mov); moveErr != nil {
			return nil, moveErr
		}
	}

	return saved, nil
}
