//go:build integration

package overhead_allocation_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestOverheadAllocationIsIsolatedByTenant: as regras de rateio de despesas
// indiretas não tinham empresa; a lista e a leitura mostravam as de todas.
func TestOverheadAllocationIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	cc := int32(1_740_000 + testutil.UniqueCode()%100_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM overhead_allocation_targets WHERE overhead_code IN (SELECT code FROM overhead_allocations WHERE enterprise_id = ANY($1))`, []int64{a.ID, b.ID})
		_, _ = pool.Exec(base, `DELETE FROM overhead_allocations WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		_, _ = pool.Exec(base, `DELETE FROM cost_centers WHERE code IN ($1, $2)`, cc, cc+1)
	})
	actor := testutil.Actor(t, pool)
	for i, e := range []testutil.Empresa{a, b} {
		if _, err := pool.Exec(base, `INSERT INTO cost_centers (code, description, enterprise_id, created_by, start_date, type) VALUES ($1,'CC',$2,$3,DATE '2026-01-01','ADMINISTRATIVE')`, cc+int32(i), e.ID, actor); err != nil {
			t.Fatal(err)
		}
	}
	repo := overhead_allocation.NewOverheadAllocationRepositorySQLC(sqlc.New(pool))
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nova := func(centro int32) *entity.OverheadAllocation {
		return &entity.OverheadAllocation{CostCenterCode: int64(centro), PeriodStart: inicio, PeriodEnd: inicio.AddDate(0, 1, -1), AllocationType: "PERCENTUAL", CreatedBy: actor}
	}
	if _, err := repo.Create(b.Ctx, nova(cc)); err == nil {
		t.Fatal("empresa B criou rateio a partir do centro de custo da A")
	}
	rat, err := repo.Create(a.Ctx, nova(cc))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByCode(b.Ctx, rat.Code); err == nil {
		t.Fatal("empresa B leu o rateio da A")
	}
	if l, err := repo.List(b.Ctx); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d rateio(s) da A (%v)", len(l), err)
	}
	if _, err := repo.AddTarget(b.Ctx, &entity.AllocationTarget{OverheadCode: rat.Code, CostCenterCode: int64(cc + 1), Percentage: 50}); err == nil {
		t.Fatal("empresa B incluiu destino no rateio da A")
	}
	if _, err := repo.AddTarget(a.Ctx, &entity.AllocationTarget{OverheadCode: rat.Code, CostCenterCode: int64(cc + 1), Percentage: 50}); err == nil {
		t.Fatal("empresa A mandou rateio para o centro de custo da B")
	}
	if _, err := repo.AddTarget(a.Ctx, &entity.AllocationTarget{OverheadCode: rat.Code, CostCenterCode: int64(cc), Percentage: 50}); err != nil {
		t.Fatalf("empresa A não incluiu destino no próprio rateio: %v", err)
	}
	if l, err := repo.GetTargets(b.Ctx, rat.Code); err != nil || len(l) != 0 {
		t.Fatalf("empresa B leu %d destino(s) do rateio da A (%v)", len(l), err)
	}
	if err := repo.Delete(b.Ctx, rat.Code); err == nil {
		t.Fatal("empresa B apagou o rateio da A")
	}
	if l, err := repo.GetTargets(a.Ctx, rat.Code); err != nil || len(l) != 1 {
		t.Fatalf("os destinos da A sumiram: %d (%v)", len(l), err)
	}
}
