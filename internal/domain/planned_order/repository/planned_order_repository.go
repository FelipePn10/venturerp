package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
)

type PlannedOrderRepository interface {
	Create(ctx context.Context, o *entity.PlannedOrder) (*entity.PlannedOrder, error)
	GetByCode(ctx context.Context, code int64) (*entity.PlannedOrder, error)
	GetByNumber(ctx context.Context, number int64) (*entity.PlannedOrder, error)
	GetByMRPSuggestionCode(ctx context.Context, suggestionCode int64) (*entity.PlannedOrder, error)
	List(ctx context.Context) ([]*entity.PlannedOrder, error)
	ListByPlan(ctx context.Context, planCode int64) ([]*entity.PlannedOrder, error)
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.PlannedOrder, error)
	ListByType(ctx context.Context, orderType string) ([]*entity.PlannedOrder, error)
	ListByStatus(ctx context.Context, status string) ([]*entity.PlannedOrder, error)
	UpdateStatus(ctx context.Context, code int64, status string) (*entity.PlannedOrder, error)
	FirmOrder(ctx context.Context, code int64) (*entity.PlannedOrder, error)
	SetPlanningState(ctx context.Context, code int64, status string, isFirm bool) (*entity.PlannedOrder, error)
	IsKanbanItem(ctx context.Context, itemCode int64) (bool, error)
	HasProductionMovements(ctx context.Context, code int64) (bool, error)
	UpdateDates(ctx context.Context, code int64, start, end *time.Time) (*entity.PlannedOrder, error)
	Delete(ctx context.Context, code int64) error
	DeleteByPlan(ctx context.Context, planCode int64) error
	GetNextOrderNumber(ctx context.Context) (int64, error)
}
