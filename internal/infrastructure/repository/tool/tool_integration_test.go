//go:build integration

package tool_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	toolrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/tool"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Exercises tool CRUD + useful-life consumption crossing the limit + replacement
// list + reset, against a real Postgres.
func TestIntegration_Tool_LifeCycle(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := toolrepo.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	tl, err := entity.NewTool(code, "Matriz Estampo", "MATRIZ", entity.LifeStrokes, 1000, 5000, uid)
	if err != nil {
		t.Fatalf("NewTool: %v", err)
	}
	created, err := repo.CreateTool(ctx, tl)
	if err != nil {
		t.Fatalf("CreateTool: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM tools WHERE id = $1", created.ID)

	// Consume 600 strokes → not yet at limit.
	after, err := repo.ConsumeToolLife(ctx, created.ID, 600)
	if err != nil {
		t.Fatalf("ConsumeToolLife: %v", err)
	}
	if after.LifeUsed != 600 || after.NeedsReplacement() {
		t.Fatalf("after 600: used=%v needs=%v, want 600 / false", after.LifeUsed, after.NeedsReplacement())
	}

	// Consume 500 more → 1100 ≥ 1000 → needs replacement.
	after, err = repo.ConsumeToolLife(ctx, created.ID, 500)
	if err != nil {
		t.Fatalf("ConsumeToolLife 2: %v", err)
	}
	if !after.NeedsReplacement() {
		t.Fatalf("after 1100/1000 should need replacement, got used=%v", after.LifeUsed)
	}

	// Appears in the replacement list.
	repl, err := repo.ListToolsNeedingReplacement(ctx)
	if err != nil {
		t.Fatalf("ListToolsNeedingReplacement: %v", err)
	}
	found := false
	for _, r := range repl {
		if r.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Error("tool over limit not in replacement list")
	}

	// Reset (after physical replacement) → life zeroed, active.
	reset, err := repo.ResetToolLife(ctx, created.ID)
	if err != nil {
		t.Fatalf("ResetToolLife: %v", err)
	}
	if reset.LifeUsed != 0 || reset.NeedsReplacement() || reset.Status != entity.StatusActive {
		t.Errorf("after reset: %+v", reset)
	}
}
