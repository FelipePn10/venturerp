package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
)

type APSRepository interface {
	UpsertSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error)
	GetSequence(ctx context.Context, id int64) (*entity.ProductionSequence, error)
	UpdateSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error)
	ListByOrder(ctx context.Context, orderID int64) ([]*entity.ProductionSequence, error)
	ListByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.ProductionSequence, error)
	DeleteByOrder(ctx context.Context, orderID int64) error

	// Data needed by the sequencing algorithm
	GetOpenProductionOrders(ctx context.Context) ([]OrderRow, error)
	GetOrderOperations(ctx context.Context, orderID int64) ([]OpRow, error)
	GetWorkCenterCapacity(ctx context.Context, workCenterID int64) (float64, error)

	// Data feeding the monthly schedule board (Gantt). [from, to) is a half-open
	// window; bars come back with their labels already joined. Bars carry raw
	// quantities/hours so the use case can derive completion and lateness.
	ListScheduledBars(ctx context.Context, from, to time.Time) ([]*entity.GanttBar, error)
	ListFallbackBars(ctx context.Context, from, to time.Time) ([]*entity.GanttBar, error)
	ListResourceLoad(ctx context.Context, from, to time.Time) ([]*entity.GanttResourceLoad, error)

	// Finish-start dependencies between scheduled bars, derived from
	// route_operation_network. ListDependencies is window-scoped (board view);
	// ListOrderDependencies returns one order's edges (cascade reschedule).
	ListDependencies(ctx context.Context, from, to time.Time) ([]*entity.GanttDependency, error)
	ListOrderDependencies(ctx context.Context, orderID int64) ([]*entity.GanttDependency, error)
}

type OrderRow struct {
	ID          int64
	Priority    int
	PlannedDate time.Time
}

type OpRow struct {
	ID           int64
	Sequence     int
	WorkCenterID *int64
	PlannedHours float64
	SetupHours   float64
}
