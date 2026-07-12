//go:build integration

package drawing_uc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/drawing_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// End-to-end drawing flow: create → revision (current) → composite code
// (Desenho(20)+Dígito+Formato+Revisão) → distribution.
func TestIntegration_Drawing_Flow(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := drawing_uc.New(q)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM enterprise ORDER BY id LIMIT 1").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()

	code := fmt.Sprintf("DES-%d-VERYLONGDRAWINGCODE-EXCEEDS-20", testutil.UniqueCode())
	d, err := uc.Create(ctx, request.DrawingDTO{Code: code, Digit: "1", Format: "A4",
		Description: "Chapa", UOM: "UN", MaterialSpec: "ACO", CreatedBy: uid})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM drawings WHERE id = $1", d.ID)

	rev, err := uc.AddRevision(ctx, d.ID, request.DrawingRevisionDTO{
		Revision: "R1", StartDate: "2026-01-01", IsCurrent: true, ApprovedBy: "Eng", UpdatedBy: uid})
	if err != nil {
		t.Fatalf("AddRevision: %v", err)
	}
	// composite = first 20 of code + digit + format + revision
	wantComposite := code[:20] + "1" + "A4" + "R1"
	if rev.CompositeCode != wantComposite {
		t.Fatalf("composite = %q, want %q", rev.CompositeCode, wantComposite)
	}
	if !rev.IsCurrent {
		t.Fatal("revisão deveria ser current")
	}

	if _, err := uc.AddDistribution(ctx, rev.ID, request.DrawingDistributionDTO{
		Recipient: "Producao", DistributedAt: "2026-01-02"}); err != nil {
		t.Fatalf("AddDistribution: %v", err)
	}

	got, err := uc.Get(ctx, d.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Revisions) != 1 || got.Revisions[0].Revision != "R1" {
		t.Fatalf("revisions = %+v, want 1 (R1)", got.Revisions)
	}
}

func TestIntegration_Drawing_TenantItemConfigurationAndRevisionReplication(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := drawing_uc.New(q)
	ctx := context.Background()
	var enterpriseID int64
	var userID uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT id FROM enterprise ORDER BY id LIMIT 1").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT created_by FROM items LIMIT 1").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	item := testutil.UniqueCode()
	plainItem := testutil.UniqueCode()
	mask := fmt.Sprintf("CFG-%d", testutil.UniqueCode())
	testutil.Exec(t, pool, "INSERT INTO items(code,warehouse_code,created_by) VALUES($1,$1,$3),($2,$2,$3)", item, plainItem, userID)
	testutil.Exec(t, pool, "INSERT INTO item_masks(item_code,mask,mask_hash,created_by,created_at) VALUES($1,$2,$3,$4,NOW())", item, mask, fmt.Sprintf("%08d", item%100000000), userID)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM drawings WHERE item_code=$1", item)
		testutil.Exec(t, pool, "DELETE FROM item_engineering_drawings WHERE item_code=$1", item)
		testutil.Exec(t, pool, "DELETE FROM item_masks WHERE item_code=$1", item)
		testutil.Exec(t, pool, "DELETE FROM item_engineering_drawings WHERE item_code=$1", plainItem)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=ANY($1::bigint[])", []int64{item, plainItem})
	})
	plain, err := uc.MaintainItemDrawingCode(ctx, request.MaintainItemDrawingCodeDTO{ItemCode: plainItem, DrawingCode: "MANUAL-PLAIN", UpdatedBy: userID})
	if err != nil || plain.Mask != "" || plain.DrawingCode != "MANUAL-PLAIN" {
		t.Fatalf("plain item=%+v err=%v", plain, err)
	}

	drawing, err := uc.Create(ctx, request.DrawingDTO{Code: "DW-REPLICATION-LONG-CODE", Digit: "-", Format: "A3", ItemCode: &item, CreatedBy: userID})
	if err != nil {
		t.Fatal(err)
	}
	r1, err := uc.AddRevision(ctx, drawing.ID, request.DrawingRevisionDTO{Revision: "R1", IsCurrent: true, UpdatedBy: userID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.GetItemDrawingCode(ctx, item, ""); err == nil {
		t.Fatal("first revision must not replicate automatically")
	}
	if _, err := uc.MaintainItemDrawingCode(ctx, request.MaintainItemDrawingCodeDTO{ItemCode: item, DrawingCode: r1.CompositeCode, UpdatedBy: userID}); err == nil {
		t.Fatal("configured item must require its mask")
	}
	for _, configuredMask := range []string{mask} {
		if _, err := uc.MaintainItemDrawingCode(ctx, request.MaintainItemDrawingCodeDTO{ItemCode: item, Mask: configuredMask, DrawingCode: r1.CompositeCode, UpdatedBy: userID}); err != nil {
			t.Fatal(err)
		}
	}
	if err := uc.UpdateManufacturingParameters(ctx, request.DrawingManufacturingParametersDTO{ReplicateDrawingRevision: true, UpdatedBy: userID}); err != nil {
		t.Fatal(err)
	}
	r2, err := uc.AddRevision(ctx, drawing.ID, request.DrawingRevisionDTO{Revision: "R2", IsCurrent: true, UpdatedBy: userID})
	if err != nil {
		t.Fatal(err)
	}
	for _, configuredMask := range []string{mask} {
		engineering, err := uc.GetItemDrawingCode(ctx, item, configuredMask)
		if err != nil || engineering.DrawingCode != r2.CompositeCode {
			t.Fatalf("mask=%q drawing=%+v err=%v", configuredMask, engineering, err)
		}
	}
	r2, err = uc.UpdateRevision(ctx, r2.ID, request.DrawingRevisionDTO{Revision: "R2B", IsCurrent: true, UpdatedBy: userID})
	if err != nil {
		t.Fatal(err)
	}
	updatedEngineering, err := uc.GetItemDrawingCode(ctx, item, mask)
	if err != nil || updatedEngineering.DrawingCode != r2.CompositeCode {
		t.Fatalf("updated revision=%+v engineering=%+v err=%v", r2, updatedEngineering, err)
	}
	if err := uc.UpdateManufacturingParameters(ctx, request.DrawingManufacturingParametersDTO{ReplicateDrawingRevision: false, UpdatedBy: userID}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddRevision(ctx, drawing.ID, request.DrawingRevisionDTO{Revision: "R3", IsCurrent: true, UpdatedBy: userID}); err != nil {
		t.Fatal(err)
	}
	engineering, err := uc.GetItemDrawingCode(ctx, item, mask)
	if err != nil || engineering.DrawingCode != r2.CompositeCode {
		t.Fatalf("parameter off must retain R2: %+v %v", engineering, err)
	}
	var otherEnterpriseID int64
	otherCode := int32(testutil.UniqueCode() % 1_000_000_000)
	if err := pool.QueryRow(ctx, "INSERT INTO enterprise(code,name) VALUES($1,'drawing isolation') RETURNING id", otherCode).Scan(&otherEnterpriseID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM enterprise WHERE id=$1", otherEnterpriseID) })
	otherCtx := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: otherEnterpriseID})
	if _, err := uc.Get(otherCtx, drawing.ID); err == nil {
		t.Fatal("cross-tenant drawing access must fail")
	}
	listed, err := uc.List(otherCtx, false, "")
	if err != nil || len(listed) != 0 {
		t.Fatalf("other tenant list=%+v err=%v", listed, err)
	}
}
