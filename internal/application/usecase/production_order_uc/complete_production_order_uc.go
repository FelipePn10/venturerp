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

type CompleteProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StockRepo is optional. When set together with a target warehouse on the
	// DTO, completing the order posts an IN stock movement of the produced
	// quantity for the finished item.
	StockRepo stockrepo.StockRepository
}

func (uc *CompleteProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CompleteProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	endDate, _ := time.Parse("2006-01-02", dto.EndDate)

	order, err := uc.Repo.Complete(ctx, dto.ID, endDate)
	if err != nil {
		return nil, err
	}

	// Post the finished-goods ENTRADA so the warehouse balance reflects the
	// completed production. Falls back to the planned quantity when nothing was
	// explicitly reported as produced.
	if uc.StockRepo != nil && dto.WarehouseID != nil {
		qty := order.ProducedQty
		if qty <= 0 {
			qty = order.PlannedQty
		}
		if qty > 0 {
			refType := stockentity.ReferenceTypeProductionOrder
			refCode := order.ID
			mov := &stockentity.StockMovement{
				ItemCode:      order.ItemCode,
				Mask:          order.Mask,
				WarehouseID:   *dto.WarehouseID,
				MovementType:  stockentity.MovementTypeIn,
				Quantity:      qty,
				ReferenceType: &refType,
				ReferenceCode: &refCode,
				CreatedBy:     order.CreatedBy,
			}
			if _, moveErr := uc.StockRepo.CreateMovement(ctx, mov); moveErr != nil {
				return nil, moveErr
			}
		}
	}

	return order, nil
}
