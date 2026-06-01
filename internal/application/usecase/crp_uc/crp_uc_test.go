package crp_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/crp/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/crp/repository"
)

// fakeCRPRepo implements repository.CRPRepository in memory.
type fakeCRPRepo struct {
	orders   []repository.PlannedOrderRow
	ops      map[int64][]repository.RouteOpRow // keyed by routeID
	avail    map[int64]float64                 // keyed by workCenterID
	upserted []*entity.CapacityRequirement
}

func (f *fakeCRPRepo) UpsertRequirement(ctx context.Context, req *entity.CapacityRequirement) (*entity.CapacityRequirement, error) {
	if req.AvailableHours > 0 {
		req.LoadPct = req.RequiredHours / req.AvailableHours * 100
	}
	f.upserted = append(f.upserted, req)
	return req, nil
}
func (f *fakeCRPRepo) ListByPlan(ctx context.Context, planCode int64) ([]*entity.CapacityRequirement, error) {
	return nil, nil
}
func (f *fakeCRPRepo) ListOverloadedByPlan(ctx context.Context, planCode int64) ([]*entity.CapacityRequirement, error) {
	return nil, nil
}
func (f *fakeCRPRepo) ListByWorkCenter(ctx context.Context, wc int64, from, to time.Time) ([]*entity.CapacityRequirement, error) {
	return nil, nil
}
func (f *fakeCRPRepo) DeleteByPlan(ctx context.Context, planCode int64) error { return nil }
func (f *fakeCRPRepo) GetPlannedOrdersByPlan(ctx context.Context, planCode int64) ([]repository.PlannedOrderRow, error) {
	return f.orders, nil
}
func (f *fakeCRPRepo) GetRouteOperationsByRoute(ctx context.Context, routeID int64) ([]repository.RouteOpRow, error) {
	return f.ops[routeID], nil
}
func (f *fakeCRPRepo) GetMachineAvailableHoursPerDay(ctx context.Context, wc int64) (float64, error) {
	return f.avail[wc], nil
}

func wcPtr(v int64) *int64 { return &v }

func TestCalculateCRP_LoadAndOverload(t *testing.T) {
	day := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	routeID := int64(1)
	repo := &fakeCRPRepo{
		orders: []repository.PlannedOrderRow{
			{ID: 1, ItemCode: 100, Quantity: 10, PlannedDate: day, RouteID: &routeID},
		},
		ops: map[int64][]repository.RouteOpRow{
			1: {
				{WorkCenterID: wcPtr(100), EffHours: 0.5}, // 0.5 * 10 = 5h
				{WorkCenterID: wcPtr(200), EffHours: 1.0}, // 1.0 * 10 = 10h
			},
		},
		avail: map[int64]float64{100: 8, 200: 8},
	}

	uc := New(repo)
	summary, err := uc.CalculateCRP(context.Background(), request.CalculateCRPDTO{PlanCode: 42})
	if err != nil {
		t.Fatalf("CalculateCRP error: %v", err)
	}
	if summary.TotalEntries != 2 {
		t.Errorf("TotalEntries = %d, want 2", summary.TotalEntries)
	}
	// WC100: 5/8 = 62.5% (ok); WC200: 10/8 = 125% (overloaded).
	if summary.OverloadCount != 1 {
		t.Errorf("OverloadCount = %d, want 1", summary.OverloadCount)
	}

	byWC := map[int64]*entity.CapacityRequirement{}
	for _, r := range repo.upserted {
		byWC[r.WorkCenterID] = r
	}
	if byWC[100].RequiredHours != 5 {
		t.Errorf("WC100 required = %v, want 5", byWC[100].RequiredHours)
	}
	if byWC[200].RequiredHours != 10 || byWC[200].LoadPct != 125 {
		t.Errorf("WC200 = %+v, want required 10 load 125", byWC[200])
	}
}

func TestCalculateCRP_AggregatesSameWorkCenterDay(t *testing.T) {
	day := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	routeID := int64(1)
	repo := &fakeCRPRepo{
		orders: []repository.PlannedOrderRow{
			{ID: 1, Quantity: 4, PlannedDate: day, RouteID: &routeID},
			{ID: 2, Quantity: 6, PlannedDate: day.Add(3 * time.Hour), RouteID: &routeID}, // same day
		},
		ops:   map[int64][]repository.RouteOpRow{1: {{WorkCenterID: wcPtr(100), EffHours: 1}}},
		avail: map[int64]float64{100: 100},
	}
	uc := New(repo)
	summary, err := uc.CalculateCRP(context.Background(), request.CalculateCRPDTO{PlanCode: 1})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Both orders hit WC100 on the same day → one aggregated entry of (4+6)*1 = 10h.
	if summary.TotalEntries != 1 {
		t.Fatalf("TotalEntries = %d, want 1 (aggregated)", summary.TotalEntries)
	}
	if repo.upserted[0].RequiredHours != 10 {
		t.Errorf("aggregated required = %v, want 10", repo.upserted[0].RequiredHours)
	}
}

func TestCalculateCRP_DefaultAvailabilityWhenZero(t *testing.T) {
	day := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	routeID := int64(1)
	repo := &fakeCRPRepo{
		orders: []repository.PlannedOrderRow{{ID: 1, Quantity: 1, PlannedDate: day, RouteID: &routeID}},
		ops:    map[int64][]repository.RouteOpRow{1: {{WorkCenterID: wcPtr(100), EffHours: 4}}},
		avail:  map[int64]float64{}, // no availability → defaults to 8
	}
	uc := New(repo)
	if _, err := uc.CalculateCRP(context.Background(), request.CalculateCRPDTO{PlanCode: 1}); err != nil {
		t.Fatalf("error: %v", err)
	}
	// 4h / default 8h = 50%.
	if repo.upserted[0].AvailableHours != 8 || repo.upserted[0].LoadPct != 50 {
		t.Errorf("default availability: got avail=%v load=%v, want 8 / 50", repo.upserted[0].AvailableHours, repo.upserted[0].LoadPct)
	}
}
