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
	paramsentity "github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
	paramsrepo "github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
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
	kanban  bool
	moved   bool
}

func (r *fakePORepo) SetPlanningState(_ context.Context, _ int64, status string, firm bool) (*entity.PlannedOrder, error) {
	if r.errFirm != nil {
		return nil, r.errFirm
	}
	if r.order == nil {
		return nil, errors.New("not found")
	}
	r.order.Status = types.OrderStatus(status)
	r.order.IsFirm = firm
	return r.order, nil
}
func (r *fakePORepo) IsKanbanItem(context.Context, int64) (bool, error) { return r.kanban, nil }
func (r *fakePORepo) HasProductionMovements(context.Context, int64) (bool, error) {
	return r.moved, nil
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
	if r.order == nil {
		return nil, errors.New("not found")
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

type fakeParams struct {
	paramsrepo.PlanningParamRepository
	value string
}

func (f *fakeParams) GetByNumber(context.Context, int) (*paramsentity.PlanningParam, error) {
	return &paramsentity.PlanningParam{ParamNumber: 25, Value: f.value}, nil
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
		Status:    types.StatusPlanned,
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
	warehouseCode := int64(44)
	order := &entity.PlannedOrder{
		ID:            77,
		Code:          99,
		ItemCode:      3001,
		Quantity:      5,
		OrderType:     types.OrderProduction,
		Status:        types.StatusPlanned,
		IsFirm:        false,
		CreatedBy:     uuid.New(),
		WarehouseCode: &warehouseCode,
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
	if prodRepo.created.PlannedOrderID == nil || *prodRepo.created.PlannedOrderID != 77 {
		t.Errorf("OF PlannedOrderID = %v, want 77", prodRepo.created.PlannedOrderID)
	}
	if prodRepo.created.OrderNumber != 201 {
		t.Errorf("OF OrderNumber = %d, want production sequence number 201", prodRepo.created.OrderNumber)
	}
	if prodRepo.created.WarehouseID == nil || *prodRepo.created.WarehouseID != warehouseCode {
		t.Fatalf("OF WarehouseID = %v, want %d", prodRepo.created.WarehouseID, warehouseCode)
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

func TestTransition_ReleaseAndReplanWithoutMovements(t *testing.T) {
	order := &entity.PlannedOrder{Code: 101, ItemCode: 10, Status: types.StatusPlanned, OrderType: types.OrderProduction}
	repo := &fakePORepo{order: order}
	uc := &FirmPlannedOrderUseCase{Repo: repo, Auth: &firmAuth{canRelease: true}}
	result, err := uc.ExecuteTransition(context.Background(), request.TransitionPlannedOrderDTO{OrderCodes: []int64{101}, Target: "RELEASED"})
	if err != nil || result[0].Status != string(types.StatusReleased) || result[0].IsFirm {
		t.Fatalf("release failed: result=%+v err=%v", result, err)
	}
	result, err = uc.ExecuteTransition(context.Background(), request.TransitionPlannedOrderDTO{OrderCodes: []int64{101}, Target: "PLANNED"})
	if err != nil || result[0].Status != string(types.StatusPlanned) {
		t.Fatalf("replan failed: result=%+v err=%v", result, err)
	}
}

func TestTransition_ReplanRejectsMovements(t *testing.T) {
	order := &entity.PlannedOrder{Code: 102, Status: types.StatusReleased}
	uc := &FirmPlannedOrderUseCase{Repo: &fakePORepo{order: order, moved: true}, Auth: &firmAuth{canRelease: true}}
	_, err := uc.ExecuteTransition(context.Background(), request.TransitionPlannedOrderDTO{OrderCodes: []int64{102}, Target: "PLANNED"})
	if !errors.Is(err, ErrOrderHasMovements) {
		t.Fatalf("expected movement guard, got %v", err)
	}
}

func TestTransition_KanbanRequiresParameter25(t *testing.T) {
	order := &entity.PlannedOrder{Code: 103, ItemCode: 9, Status: types.StatusPlanned}
	repo := &fakePORepo{order: order, kanban: true}
	uc := &FirmPlannedOrderUseCase{Repo: repo, Auth: &firmAuth{canRelease: true}, Params: &fakeParams{value: "N"}}
	_, err := uc.ExecuteTransition(context.Background(), request.TransitionPlannedOrderDTO{OrderCodes: []int64{103}, Target: "FIRM"})
	if !errors.Is(err, ErrKanbanReleaseDisabled) {
		t.Fatalf("expected Kanban guard, got %v", err)
	}
	uc.Params = &fakeParams{value: "S"}
	if _, err := uc.ExecuteTransition(context.Background(), request.TransitionPlannedOrderDTO{OrderCodes: []int64{103}, Target: "FIRM"}); err != nil {
		t.Fatalf("parameter 25 should allow Kanban: %v", err)
	}
}

func TestTransition_FirmRejectsDateChange(t *testing.T) {
	date := "2026-07-15"
	uc := &FirmPlannedOrderUseCase{Repo: &fakePORepo{}, Auth: &firmAuth{canRelease: true}}
	_, err := uc.ExecuteTransition(context.Background(), request.TransitionPlannedOrderDTO{OrderCodes: []int64{1}, Target: "FIRM", StartDate: &date})
	if !errors.Is(err, ErrFirmDateChange) {
		t.Fatalf("expected firm-date guard, got %v", err)
	}
}
