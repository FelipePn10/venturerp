//go:build integration

package drawing_uc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/drawing_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// End-to-end drawing flow: create → revision (current) → composite code
// (Desenho(20)+Dígito+Formato+Revisão) → distribution.
func TestIntegration_Drawing_Flow(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := drawing_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := fmt.Sprintf("DES-%d-VERYLONGDRAWINGCODE-EXCEEDS-20", testutil.UniqueCode())
	d, err := uc.Create(ctx, request.DrawingDTO{Code: code, Digit: "1", Format: "A4",
		Description: "Chapa", UOM: "UN", MaterialSpec: "ACO", CreatedBy: uid})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM drawings WHERE id = $1", d.ID)

	rev, err := uc.AddRevision(ctx, d.ID, request.DrawingRevisionDTO{
		Revision: "R1", StartDate: "2026-01-01", IsCurrent: true, ApprovedBy: "Eng"})
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
