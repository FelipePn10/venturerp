//go:build integration

package purchase_requisition_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	reqrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestIntegration_Requisition_AttendanceStatusRecompute(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := reqrepo.New(q, pool)
	ctx := context.Background()

	code, err := repo.NextCode(ctx)
	if err != nil {
		t.Fatalf("NextCode: %v", err)
	}
	req, err := entity.NewPurchaseRequisition(code, 1, uuid.New())
	if err != nil {
		t.Fatalf("NewPurchaseRequisition: %v", err)
	}
	if _, err := repo.Create(ctx, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM purchase_requisitions WHERE code = $1", code)

	item, err := repo.AddItem(ctx, &entity.PurchaseRequisitionItem{
		RequisitionCode: code, Sequence: 1, ItemCode: 999, Quantity: 100, Status: entity.ReqStatusOpen,
	})
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if item.Balance() != 100 {
		t.Fatalf("initial balance = %v, want 100", item.Balance())
	}

	// Attend 40 → PARTIAL.
	upd, err := repo.RegisterAttendance(ctx, item.ID, 40)
	if err != nil {
		t.Fatalf("RegisterAttendance 40: %v", err)
	}
	if upd.AttendedQty != 40 || upd.Status != entity.ReqStatusPartial {
		t.Errorf("after 40: attended=%v status=%s, want 40/PARTIAL", upd.AttendedQty, upd.Status)
	}
	if upd.Balance() != 60 {
		t.Errorf("balance after 40 = %v, want 60", upd.Balance())
	}

	// Attend remaining 60 → ATTENDED.
	upd, err = repo.RegisterAttendance(ctx, item.ID, 60)
	if err != nil {
		t.Fatalf("RegisterAttendance 60: %v", err)
	}
	if upd.AttendedQty != 100 || upd.Status != entity.ReqStatusAttended {
		t.Errorf("after 100: attended=%v status=%s, want 100/ATTENDED", upd.AttendedQty, upd.Status)
	}
	if upd.Balance() != 0 {
		t.Errorf("balance after full = %v, want 0", upd.Balance())
	}
}
