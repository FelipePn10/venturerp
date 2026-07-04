//go:build integration

package bom_header_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	bomheaderrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_header"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Exercises the BOM header: auto-versioning per item, listing and status change.
func TestIntegration_BomHeader_VersioningAndStatus(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := bomheaderrepo.New(q)
	ctx := context.Background()
	uid := uuid.New()

	item := testutil.UniqueCode()
	defer testutil.Exec(t, pool, "DELETE FROM bom_headers WHERE item_code = $1", item)

	// First version.
	v1, err := repo.NextVersion(ctx, item, "")
	if err != nil || v1 != 1 {
		t.Fatalf("NextVersion #1 = %d err=%v, want 1", v1, err)
	}
	h1, _ := entity.NewBomHeader(item, nil, "MBOM", v1, nil, uid)
	created, err := repo.Create(ctx, h1)
	if err != nil {
		t.Fatalf("Create v1: %v", err)
	}
	if created.Version != 1 || created.Status != entity.StatusDraft {
		t.Fatalf("created = %+v, want version 1 / DRAFT", created)
	}

	// Second version auto-increments.
	v2, _ := repo.NextVersion(ctx, item, "")
	if v2 != 2 {
		t.Fatalf("NextVersion #2 = %d, want 2", v2)
	}
	h2, _ := entity.NewBomHeader(item, nil, "MBOM", v2, nil, uid)
	if _, err := repo.Create(ctx, h2); err != nil {
		t.Fatalf("Create v2: %v", err)
	}

	list, err := repo.ListByItem(ctx, item)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListByItem = %d err=%v, want 2", len(list), err)
	}
	if list[0].Version != 2 {
		t.Errorf("list should be version-desc, got first = %d", list[0].Version)
	}

	// Approve v1.
	approved, err := repo.UpdateStatus(ctx, created.ID, entity.StatusApproved)
	if err != nil || approved.Status != entity.StatusApproved {
		t.Fatalf("UpdateStatus = %+v err=%v, want APPROVED", approved, err)
	}
}
