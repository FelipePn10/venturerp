package sales_order_uc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/shopspring/decimal"
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
	if f.order == nil {
		return &entity.SalesOrder{Code: 1, Status: entity.SalesOrderStatusDraft}, nil
	}
	return f.order, nil
}
func (f *fakeSORepo) ListItems(context.Context, int64) ([]*entity.SalesOrderItem, error) {
	return f.items, nil
}

// umaLinha é a linha mínima que deixa o pedido confirmar.
func umaLinha() []*entity.SalesOrderItem {
	return []*entity.SalesOrderItem{{Code: 1, IsActive: true, RequestedQty: decimal.NewFromInt(5)}}
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
	falha := errors.New("db down")
	repo := &fakeSORepo{changeErr: falha, items: umaLinha()}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}

	err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"})
	if !errors.Is(err, falha) {
		t.Fatalf("err = %v, esperado o erro do repositório", err)
	}
}

func TestChangeStatus_NonConfirmationOnlyChangesStatus(t *testing.T) {
	repo := &fakeSORepo{}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "OF"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.statusChanged {
		t.Fatal("status should have changed")
	}
}

// A confirmação não lê as linhas para gerar demanda: o MRP lê o pedido
// direto, e a demanda independente gerada aqui contava o pedido duas vezes.
func TestChangeStatus_ConfirmationDoesNotProjectDemand(t *testing.T) {
	repo := &fakeSORepo{items: umaLinha()}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}

	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "P"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Uma leitura só: a conferência da situação atual do pedido (cancelado e
	// faturado não voltam). Nada de reler para projetar demanda.
	if repo.getByCodeCalls != 1 {
		t.Fatalf("leituras do pedido = %d, esperado 1", repo.getByCodeCalls)
	}
}

// Pedido sem item, só com linha inativa ou com tudo cancelado não é venda: não
// vira pedido (P) nem pedido em análise (A).
func TestChangeStatus_NaoConfirmaPedidoSemItemAVender(t *testing.T) {
	casos := map[string][]*entity.SalesOrderItem{
		"sem itens":      nil,
		"linha inativa":  {{Code: 1, IsActive: false, RequestedQty: decimal.NewFromInt(5)}},
		"tudo cancelado": {{Code: 1, IsActive: true, RequestedQty: decimal.NewFromInt(5), CancelledQty: decimal.NewFromInt(5)}},
	}
	for nome, itens := range casos {
		for _, status := range []string{"P", "A"} {
			t.Run(nome+" → "+status, func(t *testing.T) {
				repo := &fakeSORepo{items: itens}
				uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}
				err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: status})
				if err == nil {
					t.Fatal("esperava a recusa da confirmação")
				}
				if repo.statusChanged {
					t.Fatal("a situação mudou mesmo sem item a vender")
				}
			})
		}
	}
}

// Com parte da linha cancelada ainda há o que vender.
func TestChangeStatus_ConfirmaComSaldoAVender(t *testing.T) {
	repo := &fakeSORepo{items: []*entity.SalesOrderItem{
		{Code: 1, IsActive: true, RequestedQty: decimal.NewFromInt(5), CancelledQty: decimal.NewFromInt(2)},
	}}
	uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}
	if err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: "A"}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !repo.statusChanged || repo.lastStatus != entity.SalesOrderStatusAnalysis {
		t.Fatalf("situação não mudou para análise: %v", repo.lastStatus)
	}
}

// O PATCH de status não cancela nem fatura, e não reabre pedido cancelado ou
// faturado: cancelar tem rotina própria e faturar é da NF-e.
func TestTrocaDeStatusNaoCancelaNemFaturaNemReabre(t *testing.T) {
	casos := []struct {
		nome   string
		atual  entity.SalesOrderStatus
		novo   string
		trecho string
	}{
		{"cancelar pelo status", entity.SalesOrderStatusOrder, "CANCELLED", "use o cancelamento"},
		{"faturar pelo status", entity.SalesOrderStatusOrder, "F", "NF-e autorizada"},
		{"status inexistente", entity.SalesOrderStatusOrder, "X", "não existe"},
		{"reabrir cancelado", entity.SalesOrderStatusCancelled, "P", "cancelado"},
		{"mexer no faturado", entity.SalesOrderStatusInvoiced, "R", "cancele a NF-e"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			repo := &fakeSORepo{order: &entity.SalesOrder{Code: 1, Status: c.atual}, items: umaLinha()}
			uc := ChangeStatusSalesOrderUseCase{Repo: repo, Auth: fakeAuth{can: true}}
			err := uc.Execute(context.Background(), request.ChangeStatusDTO{Code: 1, Status: c.novo})
			if err == nil || !strings.Contains(err.Error(), c.trecho) {
				t.Fatalf("erro = %v, esperado conter %q", err, c.trecho)
			}
			if repo.statusChanged {
				t.Fatal("o status não pode mudar")
			}
		})
	}
}
