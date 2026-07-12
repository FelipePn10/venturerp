package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/shopspring/decimal"
)

type productionMovementReader interface {
	ListMovements(context.Context) ([]*stockentity.StockMovement, error)
}

type OperationalConsultationUseCase struct {
	Repo  repository.ProductionOrderRepository
	Stock productionMovementReader
	Auth  ports.AuthService
}

func (uc *OperationalConsultationUseCase) Execute(ctx context.Context, id int64) (*response.ProductionOrderOperationalResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	order, err := uc.Repo.GetByCode(ctx, id)
	if err != nil {
		return nil, err
	}
	deliveries, err := uc.Repo.ListDeliveries(ctx, id)
	if err != nil {
		return nil, err
	}
	appointments, err := uc.Repo.GetAppointments(ctx, id)
	if err != nil {
		return nil, err
	}
	consumptions, err := uc.Repo.GetConsumptions(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &response.ProductionOrderOperationalResponse{Order: toProductionOrderResponse(order),
		Appointments: toProductionAppointmentResponses(appointments), Consumptions: toProductionConsumptionResponses(consumptions),
		Totals: map[string]decimal.Decimal{"planned": decimal.NewFromFloat(order.PlannedQty), "produced": decimal.NewFromFloat(order.ProducedQty), "scrapped": decimal.NewFromFloat(order.ScrappedQty)}}
	for _, d := range deliveries {
		result.Deliveries = append(result.Deliveries, &response.ProductionDeliveryResponse{ID: d.ID, Quantity: d.Quantity,
			MovementClass: d.MovementClass, WarehouseID: d.WarehouseID, Lot: d.Lot, Final: d.IsFinal, DeliveredAt: d.DeliveredAt})
		result.Totals["delivered"] = result.Totals["delivered"].Add(d.Quantity)
	}
	if uc.Stock != nil {
		movements, err := uc.Stock.ListMovements(ctx)
		if err != nil {
			return nil, err
		}
		for _, movement := range movements {
			if movement.ReferenceType == nil || movement.ReferenceCode == nil || *movement.ReferenceType != stockentity.ReferenceTypeProductionOrder || *movement.ReferenceCode != id {
				continue
			}
			result.Movements = append(result.Movements, &response.ProductionMovementResponse{ID: movement.ID, ItemCode: movement.ItemCode,
				WarehouseID: movement.WarehouseID, MovementType: movement.MovementType, Quantity: movement.Quantity, Lot: movement.Lot})
		}
	}
	result.Totals["pending"] = decimal.NewFromFloat(order.PlannedQty).Sub(result.Totals["delivered"])
	return result, nil
}
