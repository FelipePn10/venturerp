//go:build integration

package lot_mask_uc_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lot_mask_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// End-to-end lot-code generation: a CARACTER + SEQ_NUMERICA mask emits LT0001 and
// advances to LT0002 on the next call (sequence state persisted).
func TestIntegration_LotMask_Generate(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := lot_mask_uc.New(q)
	uc.Now = func() time.Time { return time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	uid := uuid.New()

	mask, err := uc.Create(ctx, request.LotMaskDTO{Application: "GERAL", Description: "test", CreatedBy: uid})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM lot_masks WHERE id = $1", mask.ID)

	if _, err := uc.AddPart(ctx, mask.ID, request.LotMaskPartDTO{
		Sequence: 1, PartType: "CARACTER", Value: "LT", Size: 2}); err != nil {
		t.Fatalf("AddPart caracter: %v", err)
	}
	if _, err := uc.AddPart(ctx, mask.ID, request.LotMaskPartDTO{
		Sequence: 2, PartType: "SEQ_NUMERICA", Value: "1", Size: 4}); err != nil {
		t.Fatalf("AddPart seq: %v", err)
	}

	g1, err := uc.Generate(ctx, request.GenerateLotDTO{LotMaskID: &mask.ID})
	if err != nil {
		t.Fatalf("Generate 1: %v", err)
	}
	if g1.Code != "LT0001" {
		t.Fatalf("code 1 = %q, want LT0001", g1.Code)
	}
	g2, err := uc.Generate(ctx, request.GenerateLotDTO{LotMaskID: &mask.ID})
	if err != nil {
		t.Fatalf("Generate 2: %v", err)
	}
	if g2.Code != "LT0002" {
		t.Fatalf("code 2 = %q, want LT0002 (sequência não avançou)", g2.Code)
	}

	// Resolution by context (application) must find the same mask.
	g3, err := uc.Generate(ctx, request.GenerateLotDTO{Application: "GERAL"})
	if err != nil {
		t.Fatalf("Generate by context: %v", err)
	}
	if g3.Code != "LT0003" {
		t.Fatalf("code 3 = %q, want LT0003", g3.Code)
	}
}
