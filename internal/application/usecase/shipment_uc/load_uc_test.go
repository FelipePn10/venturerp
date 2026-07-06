package shipment_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/google/uuid"
)

type fakeLoadRepo struct {
	fakeShipmentRepo

	load          *entity.ShipmentLoad
	shipment      *entity.Shipment
	linked        *entity.ShipmentLoadShipment
	transition    entity.LoadStatus
	recalcLoadHit bool
}

func (f *fakeLoadRepo) CreateLoad(ctx context.Context, in CreateLoadInput) (*entity.ShipmentLoad, error) {
	f.load = &entity.ShipmentLoad{ID: 1, Code: 9001, Status: entity.LoadStatusPlanned, CreatedBy: in.CreatedBy}
	return f.load, nil
}

func (f *fakeLoadRepo) GetLoadByCode(ctx context.Context, code int64) (*entity.ShipmentLoad, error) {
	return f.load, nil
}

func (f *fakeLoadRepo) AddShipmentToLoad(ctx context.Context, loadCode, shipmentCode int64, sequence int) (*entity.ShipmentLoadShipment, error) {
	f.linked = &entity.ShipmentLoadShipment{ID: 1, LoadCode: loadCode, ShipmentCode: shipmentCode, Sequence: sequence}
	return f.linked, nil
}

func (f *fakeLoadRepo) RecalcLoadTotals(ctx context.Context, code int64) error {
	f.recalcLoadHit = true
	return nil
}

func (f *fakeLoadRepo) UpdateLoadStatus(ctx context.Context, code int64, status entity.LoadStatus, by *uuid.UUID, note string) error {
	f.transition = status
	return nil
}

func TestCreateLoadRequiresActor(t *testing.T) {
	uc := &ShipmentUseCase{Repo: &fakeLoadRepo{}}

	_, err := uc.CreateLoad(context.Background(), CreateLoadInput{})
	if err == nil || !strings.Contains(err.Error(), "usuário") {
		t.Fatalf("expected actor validation error, got %v", err)
	}
}

func TestAddShipmentToLoadAssignsNextSequenceAndRecalculates(t *testing.T) {
	repo := &fakeLoadRepo{
		load: &entity.ShipmentLoad{Code: 9001, Status: entity.LoadStatusPlanned, TotalShipments: 2},
		fakeShipmentRepo: fakeShipmentRepo{
			getByCodeResp: &entity.Shipment{Code: 1001, Status: entity.ShipmentStatusOpen},
		},
	}
	uc := &ShipmentUseCase{Repo: repo}

	result, err := uc.AddShipmentToLoad(context.Background(), 9001, 1001, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sequence != 3 {
		t.Fatalf("sequence = %d, want 3", result.Sequence)
	}
	if !repo.recalcLoadHit {
		t.Fatal("expected load totals recalculation")
	}
}

func TestAddShipmentToShippedLoadFails(t *testing.T) {
	repo := &fakeLoadRepo{
		load: &entity.ShipmentLoad{Code: 9001, Status: entity.LoadStatusShipped},
	}
	uc := &ShipmentUseCase{Repo: repo}

	_, err := uc.AddShipmentToLoad(context.Background(), 9001, 1001, 1)
	if err == nil {
		t.Fatal("expected error adding shipment to shipped load")
	}
}

func TestReleaseLoadWithoutShipmentsFails(t *testing.T) {
	repo := &fakeLoadRepo{
		load: &entity.ShipmentLoad{Code: 9001, Status: entity.LoadStatusPlanned},
	}
	uc := &ShipmentUseCase{Repo: repo}

	err := uc.TransitionLoad(context.Background(), 9001, entity.LoadStatusReleased, uuid.New(), "")
	if err == nil || !strings.Contains(err.Error(), "não possui romaneios") {
		t.Fatalf("expected empty load validation, got %v", err)
	}
}

func TestReleaseLoadWithShipments(t *testing.T) {
	repo := &fakeLoadRepo{
		load: &entity.ShipmentLoad{
			Code:           9001,
			Status:         entity.LoadStatusPlanned,
			TotalShipments: 1,
		},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.TransitionLoad(context.Background(), 9001, entity.LoadStatusReleased, uuid.New(), "ok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.transition != entity.LoadStatusReleased {
		t.Fatalf("transition = %s, want RELEASED", repo.transition)
	}
}
