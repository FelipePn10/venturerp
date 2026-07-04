package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
)

type StandardCostRepository interface {
	// Item standard costs
	UpsertItemStandardCost(ctx context.Context, cost *entity.ItemStandardCost) (*entity.ItemStandardCost, error)
	GetItemStandardCost(ctx context.Context, itemCode int64, mask string) (*entity.ItemStandardCost, error)
	ListItemStandardCosts(ctx context.Context, itemCode int64) ([]*entity.ItemStandardCost, error)

	// Work center costs
	UpsertWorkCenterCost(ctx context.Context, wcc *entity.WorkCenterCost) (*entity.WorkCenterCost, error)
	GetWorkCenterCost(ctx context.Context, workCenterID int64) (*entity.WorkCenterCost, error)
	ListWorkCenterCosts(ctx context.Context) ([]*entity.WorkCenterCost, error)

	// Item purchase costs
	UpsertItemPurchaseCost(ctx context.Context, ipc *entity.ItemPurchaseCost) (*entity.ItemPurchaseCost, error)
	GetItemPurchaseCost(ctx context.Context, itemCode int64) (*entity.ItemPurchaseCost, error)

	// Log
	InsertRollupLog(ctx context.Context, entry *entity.CostRollupLogEntry) error

	// Read helpers used by the rollup algorithm
	GetDirectChildren(ctx context.Context, parentCode int64, mask string) ([]BOMChild, error)
	GetRouteHoursByItem(ctx context.Context, itemCode int64, mask string) (float64, error)
}

// BOMChild is a lightweight projection of an item structure row used during rollup.
type BOMChild struct {
	ChildCode          int64
	Quantity           float64
	LossPercentage     float64
	IsCoproduct        bool // OUTPUT (co-produto/sucata) → credita o custo do pai
	IsFixedQty         bool // quantidade por lote → amortizada pelo lote de referência
	SubstituteGroup    int16
	SubstitutePriority int16
}
