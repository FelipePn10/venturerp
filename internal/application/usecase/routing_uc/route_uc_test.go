package routing_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/google/uuid"
)

// ── Fake ─────────────────────────────────────────────────────────────────────

type fakeRoutingRepo struct {
	repository.RoutingRepository
	nextCode  int64
	route     *entity.ManufacturingRoute
	errCode   error
	errCreate error
}

func (r *fakeRoutingRepo) NextRouteCode(_ context.Context) (int64, error) {
	return r.nextCode, r.errCode
}
func (r *fakeRoutingRepo) CreateRoute(_ context.Context, rt *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error) {
	if r.errCreate != nil {
		return nil, r.errCreate
	}
	rt.ID = 501
	r.route = rt
	return rt, nil
}
func (r *fakeRoutingRepo) GetRouteByID(_ context.Context, id int64) (*entity.ManufacturingRoute, error) {
	if r.route != nil && r.route.ID == id {
		return r.route, nil
	}
	return nil, errors.New("not found")
}
func (r *fakeRoutingRepo) UpdateRoute(_ context.Context, rt *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error) {
	r.route = rt
	return rt, nil
}
func (r *fakeRoutingRepo) ListRoutesByItem(_ context.Context, _ int64) ([]*entity.ManufacturingRoute, error) {
	if r.route != nil {
		return []*entity.ManufacturingRoute{r.route}, nil
	}
	return nil, nil
}
func (r *fakeRoutingRepo) CreatedByFromUUID(v uuid.UUID) uuid.UUID { return v }

// ── Tests ─────────────────────────────────────────────────────────────────────

func sptr(s string) *string { return &s }

func TestRouteCreate_Success(t *testing.T) {
	repo := &fakeRoutingRepo{nextCode: 300}
	uc := NewRouteUseCase(repo)

	dto := request.CreateRouteDTO{
		ItemCode:    10001,
		Mask:        sptr("DEFAULT"),
		Alternative: 1,
		Description: sptr("Rota principal Suporte SS-100"),
		IsStandard:  true,
		CreatedBy:   uuid.New(),
	}

	result, err := uc.Create(context.Background(), dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemCode != 10001 {
		t.Errorf("ItemCode = %d, want 10001", result.ItemCode)
	}
	if result.Alternative != 1 {
		t.Errorf("Alternative = %d, want 1", result.Alternative)
	}
	if repo.route == nil {
		t.Fatal("route was not persisted")
	}
}

func TestRouteCreate_InvalidItemCode(t *testing.T) {
	uc := NewRouteUseCase(&fakeRoutingRepo{nextCode: 1})
	_, err := uc.Create(context.Background(), request.CreateRouteDTO{ItemCode: 0})
	if err == nil {
		t.Fatal("expected error for ItemCode=0")
	}
}

func TestRouteCreate_DefaultAlternative(t *testing.T) {
	repo := &fakeRoutingRepo{nextCode: 1}
	uc := NewRouteUseCase(repo)
	_, err := uc.Create(context.Background(), request.CreateRouteDTO{ItemCode: 1, Alternative: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.route.Alternative != 1 {
		t.Errorf("Alternative should default to 1 when 0 provided, got %d", repo.route.Alternative)
	}
}

func TestRouteCreate_RepoError(t *testing.T) {
	repo := &fakeRoutingRepo{nextCode: 1, errCreate: errors.New("db down")}
	uc := NewRouteUseCase(repo)
	_, err := uc.Create(context.Background(), request.CreateRouteDTO{ItemCode: 5000, Alternative: 1})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

func TestRouteUpdate_FieldPropagation(t *testing.T) {
	existing := &entity.ManufacturingRoute{ID: 501, ItemCode: 10001, Description: sptr("old")}
	repo := &fakeRoutingRepo{route: existing}
	uc := NewRouteUseCase(repo)

	dto := request.UpdateRouteDTO{ID: 501, Description: sptr("updated desc"), IsStandard: true}
	result, err := uc.Update(context.Background(), dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "updated desc"
	if result.Description == nil || *result.Description != want {
		t.Errorf("description not updated, got %v", result.Description)
	}
}

func TestRouteListByItem_ReturnsAll(t *testing.T) {
	repo := &fakeRoutingRepo{route: &entity.ManufacturingRoute{ID: 501, ItemCode: 10001}}
	uc := NewRouteUseCase(repo)

	results, err := uc.ListByItem(context.Background(), 10001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 route, got %d", len(results))
	}
}
