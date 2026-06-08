package sales_order_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	demandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	demandrepo "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/google/uuid"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeAuth struct {
	ports.AuthService
	can bool
}

func (f fakeAuth) CanUpdateSalesOrder(context.Context) bool { return f.can }

type fakeSORepo struct {
	repository.SalesOrderRepository
	order          *entity.SalesOrder
	items          []*entity.SalesOrderItem
	changeErr      error
	statusChanged  bool
	lastStatus     entity.SalesOrderStatus
	getByCodeCalls int
}

func (f *fakeSORepo) ChangeStatus(_ context.Context, _ int64, status entity.SalesOrderStatus) error {
	if f.changeErr != nil {
		return f.changeErr
	}
	f.statusChanged = true
	f.lastStatus = status
	return nil
}
func (f *fakeSORepo) GetByCode(context.Context, int64) (*entity.SalesOrder, error) {
	f.getByCodeCalls++
	return f.order, nil
}
func (f *fakeSORepo) ListItems(context.Context, int64) ([]*entity.SalesOrderItem, error) {
	return f.items, nil
}

type fakeDemandRepo struct {
	demandrepo.IndependentDemandRepository
	created []*demandentity.IndependentDemand
}

func (f *fakeDemandRepo) Create(_ context.Context, d *demandentity.IndependentDemand) (*demandentity.IndependentDemand, error) {
	f.created = append(f.created, d)
	return d, nil
}

func activeItem(seq int, item int64, qty float64) *entity.SalesOrderItem {
	return &entity.SalesOrderItem{
		Sequence: seq, ItemCode: item, RequestedQty: qty,
		Status: "OPEN", IsActive: true,
	}
}

// ─── Execute: auth + status + automation trigger ────────────────────────────

func TestChangeStatus_Unauthorized(t *testing.T) {
	repo := &fakeSORepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: false}}
	err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"})
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if repo.statusChanged {
		t.Fatal("status must not change when unauthorized")
	}
}

func TestChangeStatus_PropagatesRepoError(t *testing.T) {
	repo := &fakeSORepo{changeErr: errors.New("db down")}
	demand := &fakeDemandRepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}, DemandRepo: demand}

	err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"})
	if err == nil {
		t.Fatal("expected repo error to propagate")
	}
	if len(demand.created) != 0 {
		t.Fatal("no demand should be generated when the status change fails")
	}
}

func TestChangeStatus_NonConfirmationDoesNotGenerateDemand(t *testing.T) {
	repo := &fakeSORepo{}
	demand := &fakeDemandRepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}, DemandRepo: demand}

	// Moving to a non-"P" status must not feed the MRP.
	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "F"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.statusChanged {
		t.Fatal("status should have changed")
	}
	if len(demand.created) != 0 {
		t.Fatal("only confirmation (P) generates demand")
	}
}

func TestChangeStatus_ConfirmationWithoutDemandRepoIsSafe(t *testing.T) {
	repo := &fakeSORepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}} // no DemandRepo

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.getByCodeCalls != 0 {
		t.Fatal("without a demand repo the automation must not run")
	}
}

// ─── generateDemands: per-line projection rules ─────────────────────────────

func TestChangeStatus_GeneratesDeterministicDemandPerActiveLine(t *testing.T) {
	delivery := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	itemWithDate := activeItem(2, 1001, 5)
	itemWithDate.DeliveryDate = &delivery
	itemWithDate.Mask = "0007"

	repo := &fakeSORepo{
		order: &entity.SalesOrder{CreatedBy: uuid.New()},
		items: []*entity.SalesOrderItem{
			itemWithDate,
			activeItem(3, 1002, 2), // no item date → falls back to order/now
			func() *entity.SalesOrderItem { i := activeItem(4, 1003, 9); i.IsActive = false; return i }(), // inactive → skip
			func() *entity.SalesOrderItem {
				i := activeItem(5, 1004, 9)
				i.Status = entity.SalesOrderItemStatusCancelled
				return i
			}(), // cancelled → skip
			activeItem(6, 1005, 0), // zero qty → skip
		},
	}
	demand := &fakeDemandRepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}, DemandRepo: demand}

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 42, Status: "P"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(demand.created) != 2 {
		t.Fatalf("expected 2 demands (active, qty>0), got %d", len(demand.created))
	}

	// First line: deterministic code = 42*100000 + seq(2) = 4200002, with mask + item date.
	d0 := demand.created[0]
	if d0.CodeDemand != 4200002 {
		t.Fatalf("demand code = %d, want 4200002 (idempotent per line)", d0.CodeDemand)
	}
	if d0.ItemCode != 1001 || d0.Quantity != 5 {
		t.Fatalf("demand item/qty wrong: %+v", d0)
	}
	if d0.Mask == nil || *d0.Mask != "0007" {
		t.Fatalf("mask should be carried from the line")
	}
	if !d0.DemandDate.Equal(delivery) {
		t.Fatalf("demand date should prefer the line delivery date, got %v", d0.DemandDate)
	}

	// Second line: code = 42*100000 + 3 = 4200003.
	if demand.created[1].CodeDemand != 4200003 {
		t.Fatalf("second demand code = %d, want 4200003", demand.created[1].CodeDemand)
	}
}

func TestChangeStatus_DemandDateFallsBackToOrderDeliveryDate(t *testing.T) {
	orderDate := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	repo := &fakeSORepo{
		order: &entity.SalesOrder{CreatedBy: uuid.New(), DeliveryDate: &orderDate},
		items: []*entity.SalesOrderItem{activeItem(1, 2001, 3)}, // no line date
	}
	demand := &fakeDemandRepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}, DemandRepo: demand}

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 7, Status: "P"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(demand.created) != 1 {
		t.Fatalf("expected 1 demand, got %d", len(demand.created))
	}
	if !demand.created[0].DemandDate.Equal(orderDate) {
		t.Fatalf("demand date should fall back to the order delivery date, got %v", demand.created[0].DemandDate)
	}
	if demand.created[0].Mask != nil {
		t.Fatalf("empty line mask should yield nil demand mask")
	}
}
