//go:build integration

package purchase_requisition_test

import (
	"github.com/shopspring/decimal"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	reqrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestIntegration_Requisition_AttendanceStatusRecompute(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := reqrepo.New(q, pool)
	ctx := testutil.TenantContext(t, pool)

	code, err := repo.NextCode(ctx)
	if err != nil {
		t.Fatalf("NextCode: %v", err)
	}
	req, err := entity.NewPurchaseRequisition(code, 1, uuid.New())
	if err != nil {
		t.Fatalf("NewPurchaseRequisition: %v", err)
	}
	criada, err := repo.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	code = criada.Code
	defer testutil.Exec(t, pool, "DELETE FROM purchase_requisitions WHERE code = $1", code)

	item, err := repo.AddItem(ctx, &entity.PurchaseRequisitionItem{
		RequisitionCode: code, Sequence: 1, ItemCode: 999, Quantity: decimal.NewFromFloat(100), Status: entity.ReqStatusOpen,
	})
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if !item.Balance().Equal(decimal.NewFromFloat(100)) {
		t.Fatalf("initial balance = %v, want 100", item.Balance())
	}

	// Attend 40 → PARTIAL.
	upd, err := repo.RegisterAttendance(ctx, item.ID, decimal.NewFromFloat(40))
	if err != nil {
		t.Fatalf("RegisterAttendance 40: %v", err)
	}
	if !upd.AttendedQty.Equal(decimal.NewFromFloat(40)) || upd.Status != entity.ReqStatusPartial {
		t.Errorf("after 40: attended=%v status=%s, want 40/PARTIAL", upd.AttendedQty, upd.Status)
	}
	if !upd.Balance().Equal(decimal.NewFromFloat(60)) {
		t.Errorf("balance after 40 = %v, want 60", upd.Balance())
	}

	// Attend remaining 60 → ATTENDED.
	upd, err = repo.RegisterAttendance(ctx, item.ID, decimal.NewFromFloat(60))
	if err != nil {
		t.Fatalf("RegisterAttendance 60: %v", err)
	}
	if !upd.AttendedQty.Equal(decimal.NewFromFloat(100)) || upd.Status != entity.ReqStatusAttended {
		t.Errorf("after 100: attended=%v status=%s, want 100/ATTENDED", upd.AttendedQty, upd.Status)
	}
	if !upd.Balance().IsZero() {
		t.Errorf("balance after full = %v, want 0", upd.Balance())
	}
}

// Duas pessoas gerando requisição juntas: a numeração é do banco inteiro, e o segundo
// caía em erro de duplicidade.
func TestIntegration_RequisicoesSimultaneasNaoDisputamONumero(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := reqrepo.New(q, pool)
	ctx := testutil.TenantContext(t, pool)
	mesmo, err := repo.NextCode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const total = 16
	codigos := make([]int64, total)
	erros := make([]error, total)
	var fila sync.WaitGroup
	for i := range total {
		fila.Add(1)
		go func() {
			defer fila.Done()
			req, err := entity.NewPurchaseRequisition(mesmo, 1, uuid.New())
			if err != nil {
				erros[i] = err
				return
			}
			criada, err := repo.Create(ctx, req)
			if err != nil {
				erros[i] = err
				return
			}
			codigos[i] = criada.Code
		}()
	}
	fila.Wait()
	vistos := map[int64]bool{}
	for i := range total {
		if erros[i] != nil {
			t.Fatalf("requisição %d: %v", i, erros[i])
		}
		if vistos[codigos[i]] {
			t.Fatalf("número repetido: %d", codigos[i])
		}
		vistos[codigos[i]] = true
		testutil.Exec(t, pool, "DELETE FROM purchase_requisitions WHERE code = $1", codigos[i])
	}
}
