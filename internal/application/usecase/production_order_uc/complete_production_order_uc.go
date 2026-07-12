package production_order_uc

import (
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
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

type atomicDeliveryRepository interface {
	RegisterDeliveryWithMovements(context.Context, *entity.ProductionDelivery, []*stockentity.StockMovement) (*entity.ProductionOrder, error)
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

	order, err := uc.Repo.GetByCode(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	warehouseID := dto.WarehouseID
	if warehouseID == nil {
		warehouseID = order.WarehouseID
	}
	if warehouseID == nil {
		return nil, errorsuc.NewValidationError("warehouse_id is required when the order has no destination warehouse")
	}
	delivered, err := uc.Repo.GetDeliveredQuantity(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	plannedQty := decimal.NewFromFloat(order.PlannedQty)
	producedQty := decimal.NewFromFloat(order.ProducedQty)
	qty := plannedQty.Sub(delivered)
	if dto.Quantity == nil && order.ProducedQty > 0 {
		qty = producedQty.Sub(delivered)
	}
	if dto.Quantity != nil {
		qty = *dto.Quantity
	}
	if qty.IsNegative() {
		return nil, errorsuc.NewValidationError("quantity must be greater than or equal to zero")
	}
	key := strings.TrimSpace(dto.IdempotencyKey)
	if key == "" {
		key = fmt.Sprintf("legacy:%d:%s:%s:%t", dto.ID, dto.EndDate, qty.String(), dto.Final)
	}
	if existing, existingErr := uc.Repo.GetDeliveryByIdempotencyKey(ctx, key); existingErr == nil {
		return uc.Repo.GetByCode(ctx, existing.ProductionOrderID)
	}
	if dto.Final && qty.IsZero() {
		pending, pendingErr := uc.Repo.HasPendingServicePurchaseOrders(ctx, dto.ID)
		if pendingErr != nil {
			return nil, pendingErr
		}
		if pending {
			return nil, errorsuc.NewValidationError("production order has pending service purchase orders")
		}
	}
	treatExcess, err := uc.Repo.TreatProductionExcess(ctx)
	if err != nil {
		return nil, err
	}
	movementClass := "EP"
	lines := []entity.ProductionDeliveryLine{}
	if treatExcess {
		movementClass = "EPP"
	}
	plannedPart := qty
	excessPart := decimal.Zero
	if treatExcess && delivered.Add(qty).GreaterThan(plannedQty) {
		plannedPart = plannedQty.Sub(delivered)
		if plannedPart.IsNegative() {
			plannedPart = decimal.Zero
		}
		excessPart = qty.Sub(plannedPart)
	}
	if plannedPart.IsPositive() {
		lines = append(lines, entity.ProductionDeliveryLine{MovementClass: movementClass, Quantity: plannedPart})
	}
	if excessPart.IsPositive() {
		lines = append(lines, entity.ProductionDeliveryLine{MovementClass: "EPE", Quantity: excessPart})
		if plannedPart.IsZero() {
			movementClass = "EPE"
		}
	}
	endDate := datetime.ParseDateOrDefault(dto.EndDate, time.Now())
	delivery := &entity.ProductionDelivery{ProductionOrderID: dto.ID, Quantity: qty, IdempotencyKey: key,
		MovementClass: movementClass, WarehouseID: *warehouseID, Lot: dto.Lot, IsFinal: dto.Final,
		DeliveredAt: endDate, CreatedBy: order.CreatedBy, Lines: lines}
	var movements []*stockentity.StockMovement
	if uc.StockRepo != nil {
		if qty.IsPositive() {
			refType := stockentity.ReferenceTypeProductionOrder
			refCode := order.ID
			appendFinishedMovement := func(class string, movementQty decimal.Decimal) {
				movementQtyFloat, _ := movementQty.Float64()
				mov := &stockentity.StockMovement{
					ItemCode:      order.ItemCode,
					Mask:          order.Mask,
					WarehouseID:   *warehouseID,
					MovementType:  class,
					Quantity:      movementQtyFloat,
					ExactQuantity: movementQty,
					ReferenceType: &refType,
					ReferenceCode: &refCode,
					Lot:           dto.Lot,
					Notes:         stringPtr(class),
					CreatedBy:     order.CreatedBy,
				}
				movements = append(movements, mov)
			}
			if len(lines) == 0 {
				appendFinishedMovement(movementClass, qty)
			} else {
				for _, line := range lines {
					appendFinishedMovement(line.MovementClass, line.Quantity)
				}
			}

			// Receive the BOM's co-products / returnable scrap into stock (an IN
			// movement of coproduct_qty × produced, into the same warehouse).
			if uc.Structure != nil {
				children, cerr := uc.Structure.GetAllDirectChildren(ctx, order.ItemCode)
				if cerr == nil {
					for _, child := range structentity.SelectPrimarySubstituteComponents(children) {
						if !child.IsCoproduct || child.Quantity <= 0 {
							automatic, componentWarehouse, infoErr := uc.Repo.GetItemAutomaticIssue(ctx, child.ChildCode)
							if infoErr != nil || !automatic || child.IsCoproduct {
								continue
							}
							componentQty := decimal.NewFromFloat(child.Quantity)
							if !child.IsFixedQty {
								componentQty = componentQty.Mul(qty)
							}
							componentQty = componentQty.Mul(decimal.NewFromFloat(1 + child.LossPercentage/100))
							componentQtyFloat, _ := componentQty.Float64()
							rep := &stockentity.StockMovement{ItemCode: child.ChildCode, WarehouseID: componentWarehouse,
								MovementType: stockentity.MovementTypePlannedRequisition, Quantity: componentQtyFloat, ReferenceType: &refType,
								ExactQuantity: componentQty,
								ReferenceCode: &refCode, Notes: stringPtr("REP"), CreatedBy: order.CreatedBy}
							movements = append(movements, rep)
							continue
						}
						coQty, _ := decimal.NewFromFloat(child.Quantity).Mul(qty).Float64()
						coMov := &stockentity.StockMovement{
							ItemCode:      child.ChildCode,
							WarehouseID:   *warehouseID,
							MovementType:  stockentity.MovementTypeIn,
							Quantity:      coQty,
							ExactQuantity: decimal.NewFromFloat(child.Quantity).Mul(qty),
							ReferenceType: &refType,
							ReferenceCode: &refCode,
							CreatedBy:     order.CreatedBy,
						}
						movements = append(movements, coMov)
					}
				}
			}
		}
	}
	if len(movements) > 0 {
		atomicRepo, ok := uc.Repo.(atomicDeliveryRepository)
		if ok {
			return atomicRepo.RegisterDeliveryWithMovements(ctx, delivery, movements)
		}
		// Compatibility path for in-memory adapters. The PostgreSQL production
		// adapter always implements atomicDeliveryRepository.
		for _, movement := range movements {
			if _, moveErr := uc.StockRepo.CreateMovement(ctx, movement); moveErr != nil {
				return nil, moveErr
			}
		}
	}
	order, err = uc.Repo.RegisterDelivery(ctx, delivery)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func stringPtr(value string) *string { return &value }
