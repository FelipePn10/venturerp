package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

type ProductionOrderRepository interface {
	Create(ctx context.Context, o *entity.ProductionOrder) (*entity.ProductionOrder, error)
	Update(ctx context.Context, o *entity.ProductionOrder) (*entity.ProductionOrder, error)
	GetByCode(ctx context.Context, id int64) (*entity.ProductionOrder, error)
	List(ctx context.Context) ([]*entity.ProductionOrder, error)
	CreateFromPlannedOrder(ctx context.Context, plannedOrderID int64) (*entity.ProductionOrder, error)
	Start(ctx context.Context, id int64, startDate time.Time) (*entity.ProductionOrder, error)
	AddAppointment(ctx context.Context, a *entity.ProductionAppointment) (*entity.ProductionAppointment, error)
	AddConsumption(ctx context.Context, c *entity.ProductionConsumption) (*entity.ProductionConsumption, error)
	Complete(ctx context.Context, id int64, endDate time.Time) (*entity.ProductionOrder, error)
	Close(ctx context.Context, id int64) (*entity.ProductionOrder, error)
	Cancel(ctx context.Context, id int64) (*entity.ProductionOrder, error)
	GetAppointments(ctx context.Context, productionOrderID int64) ([]*entity.ProductionAppointment, error)
	GetConsumptions(ctx context.Context, productionOrderID int64) ([]*entity.ProductionConsumption, error)
	GetNextOrderNumber(ctx context.Context) (int64, error)

	// Cost settlement (custo real da OF).
	// ComputeActualCostInputs aggregates the shop-floor actuals: material valued
	// at the consumed items' weighted-average cost and labor from the appointed
	// hours × the work center's cost/hour.
	ComputeActualCostInputs(ctx context.Context, productionOrderID int64) (materialReal, laborReal float64, err error)
	SettleCost(ctx context.Context, c *entity.ProductionOrderCost) (*entity.ProductionOrderCost, error)
	GetCost(ctx context.Context, productionOrderID int64) (*entity.ProductionOrderCost, error)
}
