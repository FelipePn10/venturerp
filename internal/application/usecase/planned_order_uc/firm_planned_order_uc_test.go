package planned_order_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakePORepo struct {
	repository.PlannedOrderRepository
	order   *entity.PlannedOrder
	errFirm error
	errGet  error
}

func (r *fakePORepo) FirmOrder(_ context.Context, code int64) (*entity.PlannedOrder, error) {
	if r.errFirm != nil {
		return nil, r.errFirm
	}
	if r.order != nil {
		r.order.IsFirm = true
		return r.order, nil
	}
	return nil, errors.New("not found")
}
func (r *fakePORepo) GetByCode(_ context.Context, _ int64) (*entity.PlannedOrder, error) {
	if r.errGet != nil {
		return nil, r.errGet
	}
	return r.order, nil
}

type fakeProdOrderRepo struct {
	productionrepo.ProductionOrderRepository
	created   *productionentity.ProductionOrder
	nextNum   int64
	errCreate error
}

func (r *fakeProdOrderRepo) GetNextOrderNumber(_ context.Context) (int64, error) {
	return r.nextNum, nil
}
func (r *fakeProdOrderRepo) Create(_ context.Context, of *productionentity.ProductionOrder) (*productionentity.ProductionOrder, error) {
	if r.errCreate != nil {
		return nil, r.errCreate
	}
	r.created = of
	return of, nil
}

type firmAuth struct {
	ports.AuthService
	canRelease bool
}

func (a *firmAuth) CanReleaseOrder(_ context.Context) bool      { return a.canRelease }
func (a *firmAuth) UserID(_ context.Context) (uuid.UUID, error) { return uuid.Nil, nil }

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestFirmOrder_Success(t *testing.T) {
	order := &entity.PlannedOrder{
		Code:      88,
		ItemCode:  5000,
		Quantity:  10,
		OrderType: types.OrderPurchase,
		IsFirm:    false,
		IsActive:  true,
	}
	repo := &fakePORepo{order: order}
	uc := &FirmPlannedOrderUseCase{Repo: repo, Auth: &firmAuth{canRelease: true}}

	result, err := uc.Execute(context.Background(), request.FirmOrderDTO{OrderCode: 88})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsFirm {
		t.Error("order must be firm after Execute")
	}
}

func TestFirmOrder_Unauthorized(t *testing.T) {
	uc := &FirmPlannedOrderUseCase{
		Repo: &fakePORepo{},
		Auth: &firmAuth{canRelease: false},
	}
	_, err := uc.Execute(context.Background(), request.FirmOrderDTO{OrderCode: 1})
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestFirmOrder_RepoError(t *testing.T) {
	uc := &FirmPlannedOrderUseCase{
		Repo: &fakePORepo{errFirm: errors.New("db error")},
		Auth: &firmAuth{canRelease: true},
	}
	_, err := uc.Execute(context.Background(), request.FirmOrderDTO{OrderCode: 1})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

func TestFirmOrder_ProductionOrderCreatedOnFirstFirm(t *testing.T) {
	order := &entity.PlannedOrder{
		Code:      99,
		ItemCode:  3001,
		Quantity:  5,
		OrderType: types.OrderProduction,
		IsFirm:    false,
		CreatedBy: uuid.New(),
	}
	poRepo := &fakePORepo{order: order}
	prodRepo := &fakeProdOrderRepo{nextNum: 201}

	uc := &FirmPlannedOrderUseCase{
		Repo:          poRepo,
		Auth:          &firmAuth{canRelease: true},
		ProdOrderRepo: prodRepo,
	}
	_, err := uc.Execute(context.Background(), request.FirmOrderDTO{OrderCode: 99})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prodRepo.created == nil {
		t.Fatal("production order was not created for PRODUCTION planned order")
	}
	if prodRepo.created.ItemCode != 3001 {
		t.Errorf("OF ItemCode = %d, want 3001", prodRepo.created.ItemCode)
	}
	if prodRepo.created.PlannedOrderID == nil || *prodRepo.created.PlannedOrderID != 99 {
		t.Errorf("OF PlannedOrderID = %v, want 99", prodRepo.created.PlannedOrderID)
	}
}

func TestFirmOrder_ProductionOrderSkippedIfAlreadyFirm(t *testing.T) {
	order := &entity.PlannedOrder{
		Code:      100,
		OrderType: types.OrderProduction,
		IsFirm:    true, // already firm
	}
	prodRepo := &fakeProdOrderRepo{nextNum: 1}

	uc := &FirmPlannedOrderUseCase{
		Repo:          &fakePORepo{order: order},
		Auth:          &firmAuth{canRelease: true},
		ProdOrderRepo: prodRepo,
	}
	_, err := uc.Execute(context.Background(), request.FirmOrderDTO{OrderCode: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prodRepo.created != nil {
		t.Error("no production order should be created when order was already firm")
	}
}
