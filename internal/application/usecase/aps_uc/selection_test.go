package aps_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	apsentity "github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

type selectionAPSRepo struct {
	*fakeAPSRepo
	filter  apsrepo.SequenceFilter
	upserts []*apsentity.ProductionSequence
}

func (f *selectionAPSRepo) GetSelectedProductionOrders(_ context.Context, filter apsrepo.SequenceFilter) ([]apsrepo.OrderRow, error) {
	f.filter = filter
	return []apsrepo.OrderRow{{ID: 11, Priority: 1, PlannedDate: mon}}, nil
}
func (f *selectionAPSRepo) GetSelectedOrderOperations(_ context.Context, _ int64, filter apsrepo.SequenceFilter) ([]apsrepo.OpRow, error) {
	f.filter = filter
	wc := int64(7)
	return []apsrepo.OpRow{{ID: 22, Sequence: 10, WorkCenterID: &wc, PlannedHours: 2}}, nil
}
func (f *selectionAPSRepo) UpsertSequence(_ context.Context, seq *apsentity.ProductionSequence) (*apsentity.ProductionSequence, error) {
	f.upserts = append(f.upserts, seq)
	return seq, nil
}
func (f *selectionAPSRepo) ListSequencingEvents(context.Context, apsrepo.SequenceFilter) ([]apsrepo.SequencingEventRow, error) {
	return []apsrepo.SequencingEventRow{{EventType: "SCRAP", ProductionOrderID: 11, OrderNumber: 101, EventAt: mon, Quantity: "2.000000"}}, nil
}
func (f *selectionAPSRepo) ListSequencingResources(context.Context) ([]apsrepo.SequencingResourceRow, error) {
	group := int64(3)
	return []apsrepo.SequencingResourceRow{{ID: 5, Code: 500, Name: "Laser", WorkCenterID: 7, ResourceGroupID: &group, IsActive: true}}, nil
}
func (f *selectionAPSRepo) ListSequencingView(context.Context, apsrepo.SequencingViewFilter) ([]*apsentity.ProductionSequence, error) {
	return []*apsentity.ProductionSequence{{ID: 9, ProductionOrderID: 11, WorkCenterID: 7, ScheduledStart: mon, ScheduledEnd: mon.Add(time.Hour), Status: apsentity.StatusScheduled}}, nil
}
func (f *selectionAPSRepo) ListAvailabilityWindows(context.Context, int64, []int64, time.Time, time.Time) ([]apsrepo.AvailabilityWindow, error) {
	return nil, nil
}
func (f *selectionAPSRepo) ListCandidateMachines(context.Context, int64, []int64) ([]apsrepo.MachineCandidate, error) {
	return nil, nil
}
func (f *selectionAPSRepo) ListMachineDowntimeWindows(context.Context, int64, time.Time, time.Time) ([]apsrepo.AvailabilityWindow, error) {
	return nil, nil
}

func TestSequenceOrdersAppliesSelection(t *testing.T) {
	repo := &selectionAPSRepo{fakeAPSRepo: &fakeAPSRepo{capacity: map[int64]float64{7: 8}}}
	uc := New(repo)
	result, err := uc.SequenceOrders(context.Background(), request.SequenceOrdersDTO{StartFrom: mon, OrderIDs: []int64{11}, MachineIDs: []int64{5}, WorkCenterIDs: []int64{7}, OperationIDs: []int64{22}})
	if err != nil {
		t.Fatal(err)
	}
	if result.OrdersProcessed != 1 || result.ScheduledOperations != 1 || len(repo.upserts) != 1 {
		t.Fatalf("result=%+v upserts=%d", result, len(repo.upserts))
	}
	if repo.filter.MachineIDs[0] != 5 || repo.filter.OperationIDs[0] != 22 {
		t.Fatalf("filter not propagated: %+v", repo.filter)
	}
}

func TestSequencingExportsAndResources(t *testing.T) {
	repo := &selectionAPSRepo{fakeAPSRepo: &fakeAPSRepo{}}
	uc := New(repo)
	events, err := uc.ExportSequencingEvents(context.Background(), request.SequenceOrdersDTO{})
	if err != nil || len(events) != 1 || events[0].EventType != "SCRAP" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	resources, err := uc.ListSequencingResources(context.Background())
	if err != nil || len(resources) != 1 || resources[0].ResourceGroupID == nil {
		t.Fatalf("resources=%+v err=%v", resources, err)
	}
}

func TestViewSequencingRequiresGroupAndValidRange(t *testing.T) {
	repo := &selectionAPSRepo{fakeAPSRepo: &fakeAPSRepo{}}
	uc := New(repo)
	if _, err := uc.ViewSequencing(context.Background(), request.SequencingViewDTO{From: mon, To: mon.Add(time.Hour)}); err == nil {
		t.Fatal("missing group must fail")
	}
	rows, err := uc.ViewSequencing(context.Background(), request.SequencingViewDTO{From: mon, To: mon.Add(time.Hour), ResourceGroupID: 3, TimeUnit: "MINUTE", RefreshValue: 12})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}
