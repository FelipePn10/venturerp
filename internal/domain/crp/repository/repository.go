package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/crp/entity"
)

type CRPRepository interface {
	UpsertRequirement(ctx context.Context, req *entity.CapacityRequirement) (*entity.CapacityRequirement, error)
	ListByPlan(ctx context.Context, planCode int64) ([]*entity.CapacityRequirement, error)
	ListOverloadedByPlan(ctx context.Context, planCode int64) ([]*entity.CapacityRequirement, error)
	ListByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.CapacityRequirement, error)
	DeleteByPlan(ctx context.Context, planCode int64) error

	// Read helpers used by the CRP calculation service
	GetPlannedOrdersByPlan(ctx context.Context, planCode int64) ([]PlannedOrderRow, error)
	GetRouteOperationsByRoute(ctx context.Context, routeID int64) ([]RouteOpRow, error)
	GetMachineAvailableHoursPerDay(ctx context.Context, workCenterID int64) (float64, error)
}

type PlannedOrderRow struct {
	MachineWorkCenterID *int64
	MachineHours        float64
	ID                  int64
	ItemCode            int64
	Quantity            float64
	PlannedDate         time.Time
	RouteID             *int64
}

type RouteOpRow struct {
	WorkCenterID *int64
	EffHours     float64
}

// ScheduledLoad is an exact reservation created by finite machine planning.
type ScheduledLoad struct {
	WorkCenterID int64
	Day          time.Time
	Hours        float64
}
