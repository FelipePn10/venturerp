package sales_order_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
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
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}

	err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"})
	if err == nil {
		t.Fatal("expected repo error to propagate")
	}
}

func TestChangeStatus_NonConfirmationOnlyChangesStatus(t *testing.T) {
	repo := &fakeSORepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "F"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.statusChanged {
		t.Fatal("status should have changed")
	}
}

// A confirmação não lê as linhas para gerar demanda: o MRP lê o pedido
// direto, e a demanda independente gerada aqui contava o pedido duas vezes.
func TestChangeStatus_ConfirmationDoesNotProjectDemand(t *testing.T) {
	repo := &fakeSORepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.getByCodeCalls != 0 {
		t.Fatal("confirmation must not read the order to project demand")
	}
}
