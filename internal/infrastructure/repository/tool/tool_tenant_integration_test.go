//go:build integration

package tool_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	toolrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/tool"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestToolsAreIsolatedByTenant: ferramentas e séries não tinham empresa.
func TestToolsAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tools WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	repo := toolrepo.New(sqlc.New(pool))
	tl, err := entity.NewTool(testutil.UniqueCode(), "MATRIZ DA A", "MATRIZ", entity.LifeStrokes, 1000, 10, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	ferr, err := repo.CreateTool(a.Ctx, tl)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetTool(b.Ctx, ferr.ID); err == nil {
		t.Fatal("empresa B leu a ferramenta da A")
	}
	if l, err := repo.ListTools(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d ferramenta(s) da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateTool(b.Ctx, ferr); err == nil {
		t.Fatal("empresa B alterou a ferramenta da A")
	}
	if _, err := repo.ResetToolLife(b.Ctx, ferr.ID); err == nil {
		t.Fatal("empresa B zerou a vida da ferramenta da A")
	}
	if err := repo.DeactivateTool(b.Ctx, ferr.ID); err == nil {
		t.Fatal("empresa B desativou a ferramenta da A")
	}
	if _, err := repo.CreateToolSerial(b.Ctx, &entity.ToolSerial{ToolID: ferr.ID, SerialNumber: "S-B", Status: entity.SerialActive}); err == nil {
		t.Fatal("empresa B criou série na ferramenta da A")
	}
	serie, err := repo.CreateToolSerial(a.Ctx, &entity.ToolSerial{ToolID: ferr.ID, SerialNumber: "S-A", Status: entity.SerialActive})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetToolSerial(b.Ctx, serie.ID); err == nil {
		t.Fatal("empresa B leu a série da A")
	}
	if _, err := repo.ListToolSerials(b.Ctx, ferr.ID, false); err == nil {
		t.Fatal("empresa B listou as séries da ferramenta da A")
	}
	if l, err := repo.ListToolsNeedingReplacement(b.Ctx, 0); err != nil || len(l) != 0 {
		t.Fatalf("empresa B viu %d ferramenta(s) da A para troca (%v)", len(l), err)
	}
	if got, err := repo.GetTool(a.Ctx, ferr.ID); err != nil || !got.IsActive {
		t.Fatalf("a ferramenta da A mudou: %+v (%v)", got, err)
	}
}
