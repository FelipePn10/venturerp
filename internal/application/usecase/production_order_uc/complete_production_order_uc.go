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
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// coproductReader exposes the BOM's direct children so completion can receive
// co-products / returnable scrap into stock.
type coproductReader interface {
	GetAllDirectChildren(ctx context.Context, parentCode int64) ([]*structentity.ItemStructure, error)
}

type CompleteProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StockRepo is optional. When set together with a target warehouse on the
	// DTO, completing the order posts an IN stock movement of the produced
	// quantity for the finished item.
	StockRepo stockrepo.StockRepository
	// Structure is optional. When set with StockRepo, completing the order also
	// receives the item's BOM co-products / returnable scrap into stock.
	Structure coproductReader
}

func (uc *CompleteProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CompleteProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ID == 0 {
		return nil, errorsuc.NewValidationError("id is required")
	}

	endDate := datetime.ParseDateOrDefault(dto.EndDate, time.Now())

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
				Lot:           dto.Lot,
				CreatedBy:     order.CreatedBy,
			}
			if _, moveErr := uc.StockRepo.CreateMovement(ctx, mov); moveErr != nil {
				return nil, moveErr
			}

			// Receive the BOM's co-products / returnable scrap into stock (an IN
			// movement of coproduct_qty × produced, into the same warehouse).
			if uc.Structure != nil {
				children, cerr := uc.Structure.GetAllDirectChildren(ctx, order.ItemCode)
				if cerr == nil {
					for _, child := range structentity.SelectPrimarySubstituteComponents(children) {
						if !child.IsCoproduct || child.Quantity <= 0 {
							continue
						}
						coQty := child.Quantity * qty
						coMov := &stockentity.StockMovement{
							ItemCode:      child.ChildCode,
							WarehouseID:   *dto.WarehouseID,
							MovementType:  stockentity.MovementTypeIn,
							Quantity:      coQty,
							ReferenceType: &refType,
							ReferenceCode: &refCode,
							CreatedBy:     order.CreatedBy,
						}
						if _, moveErr := uc.StockRepo.CreateMovement(ctx, coMov); moveErr != nil {
							return nil, moveErr
						}
					}
				}
			}
		}
	}

	return order, nil
}
