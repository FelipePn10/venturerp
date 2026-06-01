//go:build integration

package entry_operation_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
	eorepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/entry_operation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestIntegration_EntryOperation_StateGroupAndOp(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := eorepo.New(q, pool)
	ctx := context.Background()

	// State group with PR and SC.
	gCode, err := repo.NextStateGroupCode(ctx)
	if err != nil {
		t.Fatalf("NextStateGroupCode: %v", err)
	}
	g, err := entity.NewStateGroup(gCode, "Sul", uuid.New())
	if err != nil {
		t.Fatalf("NewStateGroup: %v", err)
	}
	if _, err := repo.CreateStateGroup(ctx, g); err != nil {
		t.Fatalf("CreateStateGroup: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM state_groups WHERE code = $1", gCode)

	for _, uf := range []string{"PR", "SC"} {
		if err := repo.AddStateGroupUF(ctx, gCode, uf); err != nil {
			t.Fatalf("AddStateGroupUF %s: %v", uf, err)
		}
	}
	// Idempotent (ON CONFLICT DO NOTHING).
	if err := repo.AddStateGroupUF(ctx, gCode, "PR"); err != nil {
		t.Fatalf("AddStateGroupUF duplicate: %v", err)
	}

	in, err := repo.UFInGroup(ctx, gCode, "PR")
	if err != nil || !in {
		t.Errorf("UFInGroup(PR) = %v err=%v, want true", in, err)
	}
	in, err = repo.UFInGroup(ctx, gCode, "RS")
	if err != nil || in {
		t.Errorf("UFInGroup(RS) = %v err=%v, want false", in, err)
	}
	ufs, err := repo.ListStateGroupUFs(ctx, gCode)
	if err != nil || len(ufs) != 2 {
		t.Errorf("ListStateGroupUFs = %v err=%v, want 2", ufs, err)
	}

	// Entry operation type referencing the group.
	oCode, err := repo.NextEntryOperationCode(ctx)
	if err != nil {
		t.Fatalf("NextEntryOperationCode: %v", err)
	}
	o, err := entity.NewEntryOperationType(oCode, "Compra dentro do estado", "1102", uuid.New())
	if err != nil {
		t.Fatalf("NewEntryOperationType: %v", err)
	}
	o.StateGroupCode = &gCode
	if _, err := repo.CreateEntryOperation(ctx, o); err != nil {
		t.Fatalf("CreateEntryOperation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM entry_operation_types WHERE code = $1", oCode)

	got, err := repo.GetEntryOperationByCode(ctx, oCode)
	if err != nil {
		t.Fatalf("GetEntryOperationByCode: %v", err)
	}
	if got.NatureOperation != "1102" || got.StateGroupCode == nil || *got.StateGroupCode != gCode {
		t.Errorf("unexpected entry op: %+v", got)
	}

	// End-to-end rule: nature 1 + UF in group → valid; UF outside → invalid.
	inGroup, _ := repo.UFInGroup(ctx, gCode, "PR")
	if err := got.ValidateUF("PR", inGroup); err != nil {
		t.Errorf("PR should be valid for nature 1 in-group: %v", err)
	}
	outGroup, _ := repo.UFInGroup(ctx, gCode, "RS")
	if err := got.ValidateUF("RS", outGroup); err == nil {
		t.Error("RS should be invalid for nature 1 (not in group)")
	}
}
