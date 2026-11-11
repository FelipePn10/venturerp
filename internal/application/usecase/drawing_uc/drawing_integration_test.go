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
	if err := pool.QueryRow(ctx, "SELECT id FROM users WHERE $1::bigint>0 ORDER BY id LIMIT 1", enterpriseID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	item := testutil.UniqueCode()
	plainItem := testutil.UniqueCode()
	mask := fmt.Sprintf("CFG-%d", testutil.UniqueCode())
	testutil.Exec(t, pool, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$1,$3,$4),($2,($2::bigint)::text,$2,$3,$4)", item, plainItem, userID, enterpriseID)
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

// O desenho é da empresa, e o que se pendura nele também: a característica e a
// resposta do configurador vêm da MESMA empresa (o configurador é por empresa)
// e o código de desenho só vale para item da empresa. Antes a empresa B
// vinculava a característica da A pelo id e gravava código de desenho para o
// item da A.
func TestIntegration_Drawing_VinculosDaMesmaEmpresa(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := drawing_uc.New(q)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM drawing_characteristics WHERE drawing_id IN (SELECT id FROM drawings WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM drawings WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM item_engineering_drawings WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM cfg_variables WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM cfg_sets WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM cfg_characteristics WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})

	caracteristica := func(e testutil.Empresa) (charID, varID int64) {
		if err := pool.QueryRow(base, `INSERT INTO cfg_characteristics(code,description,char_type,created_by,enterprise_id)
			VALUES($1,'Cor','ESCOLHA',$2,$3) RETURNING id`, fmt.Sprintf("C%d", testutil.UniqueCode()), actor, e.ID).Scan(&charID); err != nil {
			t.Fatal(err)
		}
		var setID int64
		if err := pool.QueryRow(base, `INSERT INTO cfg_sets(description,created_by,enterprise_id) VALUES('Cores',$1,$2) RETURNING id`, actor, e.ID).Scan(&setID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(base, `INSERT INTO cfg_variables(set_id,code,description,mask_composition,created_by,enterprise_id)
			VALUES($1,'AZ','Azul','AZ',$2,$3) RETURNING id`, setID, actor, e.ID).Scan(&varID); err != nil {
			t.Fatal(err)
		}
		return charID, varID
	}
	charA, varA := caracteristica(a)
	charB, varB := caracteristica(b)

	desenhoB, err := uc.Create(b.Ctx, request.DrawingDTO{Code: fmt.Sprintf("DB-%d", testutil.UniqueCode()), Description: "Chapa", CreatedBy: actor})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := uc.AddCharacteristic(b.Ctx, desenhoB.ID, request.DrawingCharacteristicDTO{CharacteristicID: charA}); err == nil {
		t.Fatal("a empresa B vinculou ao desenho dela a característica da empresa A")
	}
	if _, err := uc.AddCharacteristic(b.Ctx, desenhoB.ID, request.DrawingCharacteristicDTO{CharacteristicID: charB, VariableID: &varA}); err == nil {
		t.Fatal("a empresa B vinculou ao desenho dela a resposta da empresa A")
	}
	if _, err := uc.AddCharacteristic(b.Ctx, desenhoB.ID, request.DrawingCharacteristicDTO{CharacteristicID: charB, Operator: "EQ"}); err == nil {
		t.Fatal("condição fora da lista do banco deveria ser recusada antes de gravar")
	}
	vinculo, err := uc.AddCharacteristic(b.Ctx, desenhoB.ID, request.DrawingCharacteristicDTO{CharacteristicID: charB, VariableID: &varB, Operator: "different"})
	if err != nil {
		t.Fatalf("vínculo da própria empresa recusado: %v", err)
	}
	if vinculo.Operator != "DIFFERENT" {
		t.Fatalf("operador gravado %q, esperado DIFFERENT", vinculo.Operator)
	}

	itemA := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, itemA, actor)
	if _, err := uc.MaintainItemDrawingCode(b.Ctx, request.MaintainItemDrawingCodeDTO{ItemCode: itemA, DrawingCode: "X-1", UpdatedBy: actor}); err == nil {
		t.Fatal("a empresa B gravou código de desenho para o item da empresa A")
	}
	if _, err := uc.MaintainItemDrawingCode(a.Ctx, request.MaintainItemDrawingCodeDTO{ItemCode: itemA, DrawingCode: "X-1", UpdatedBy: actor}); err != nil {
		t.Fatalf("a dona do item não gravou o código de desenho: %v", err)
	}
}
