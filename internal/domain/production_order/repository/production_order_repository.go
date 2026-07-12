package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	RegisterDelivery(ctx context.Context, delivery *entity.ProductionDelivery) (*entity.ProductionOrder, error)
	GetDeliveryByIdempotencyKey(ctx context.Context, key string) (*entity.ProductionDelivery, error)
	HasPendingServicePurchaseOrders(ctx context.Context, productionOrderID int64) (bool, error)
	TreatProductionExcess(ctx context.Context) (bool, error)
	GetDeliveredQuantity(ctx context.Context, productionOrderID int64) (decimal.Decimal, error)
	GetItemAutomaticIssue(ctx context.Context, itemCode int64) (automatic bool, warehouseID int64, err error)
	ListDeliveries(ctx context.Context, productionOrderID int64) ([]*entity.ProductionDelivery, error)

	// Cost settlement (custo real da OF).
	// ComputeActualCostInputs aggregates the shop-floor actuals: material valued
	// at the consumed items' weighted-average cost and labor from the appointed
	// hours × the work center's cost/hour.
	ComputeActualCostInputs(ctx context.Context, productionOrderID int64) (materialReal, laborReal float64, err error)
	SettleCost(ctx context.Context, c *entity.ProductionOrderCost) (*entity.ProductionOrderCost, error)
	GetCost(ctx context.Context, productionOrderID int64) (*entity.ProductionOrderCost, error)

	ListMaterials(ctx context.Context, productionOrderID int64, kind entity.MaterialKind) ([]*entity.ProductionOrderMaterial, error)
	AddMaterial(ctx context.Context, material *entity.ProductionOrderMaterial) (*entity.ProductionOrderMaterial, error)
	ReplaceMaterial(ctx context.Context, materialID int64, replacements []entity.MaterialSubstitution, createdBy uuid.UUID) ([]*entity.ProductionOrderMaterial, error)
	DeleteMaterial(ctx context.Context, materialID int64) error
	HasActiveWMSRequest(ctx context.Context, materialID int64) (bool, error)
	AllocateLots(ctx context.Context, materialID int64, movementKind string, allocations []entity.LotAllocation, createdBy uuid.UUID) ([]entity.LotAllocation, error)
	AllocateLotsBatch(ctx context.Context, materialIDs []int64, movementKind string, lots []entity.LotAllocation, createdBy uuid.UUID) ([]entity.LotAllocation, error)
	AddScrapDestination(ctx context.Context, destination *entity.ScrapDestination) (*entity.ScrapDestination, error)
	HasProductionActivity(ctx context.Context, productionOrderID int64) (bool, error)
	CanChangeOrderQuantity(ctx context.Context, productionOrderID int64) (bool, error)
	CanChangeOrderDates(ctx context.Context, productionOrderID int64) (bool, error)
	AcceptsFractionalQuantity(ctx context.Context, itemCode int64) (bool, error)
	UpsertWMSSettings(ctx context.Context, settings entity.WMSWarehouseSettings) (*entity.WMSWarehouseSettings, error)
}
