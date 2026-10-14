//go:build integration

package quality_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/quality/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/quality"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Plano de inspeção, característica, registro e não conformidade são da
// empresa do item: outra empresa não lê, não altera nem pendura nada neles.
func TestQualidadeIsoladaPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM non_conformances WHERE item_code IN (SELECT code FROM items WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM quality_measurements WHERE record_id IN (SELECT id FROM quality_records WHERE item_code IN (SELECT code FROM items WHERE enterprise_id = ANY($1)))`, ids)
		pool.Exec(base, `DELETE FROM quality_records WHERE item_code IN (SELECT code FROM items WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM inspection_plan_characteristics WHERE plan_id IN (SELECT id FROM inspection_plans WHERE item_code IN (SELECT code FROM items WHERE enterprise_id = ANY($1)))`, ids)
		pool.Exec(base, `DELETE FROM inspection_plans WHERE item_code IN (SELECT code FROM items WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	repo := quality.New(sqlc.New(pool))

	plano, err := repo.CreatePlan(a.Ctx, &entity.InspectionPlan{ItemCode: item, PointType: entity.PointShipping, Description: "Final", SampleSize: 1, AcceptanceLevel: 0, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreatePlan(b.Ctx, &entity.InspectionPlan{ItemCode: item, PointType: entity.PointShipping, Description: "Intruso", SampleSize: 1, CreatedBy: actor}); err == nil {
		t.Fatal("empresa B criou plano para o item da A")
	}
	if _, err := repo.GetPlanByID(b.Ctx, plano.ID); err == nil {
		t.Fatal("empresa B leu o plano da A")
	}
	if lista, err := repo.ListPlansByItem(b.Ctx, item); err != nil || len(lista) != 0 {
		t.Fatalf("empresa B lista planos da A: %d (%v)", len(lista), err)
	}
	if _, err := repo.AddCharacteristic(b.Ctx, &entity.InspectionCharacteristic{PlanID: plano.ID, Name: "x"}); err == nil {
		t.Fatal("empresa B pendurou característica no plano da A")
	}
	if _, err := repo.AddCharacteristic(a.Ctx, &entity.InspectionCharacteristic{PlanID: plano.ID, Name: "Diâmetro"}); err != nil {
		t.Fatal(err)
	}
	if cs, err := repo.ListCharacteristics(b.Ctx, plano.ID); err != nil || len(cs) != 0 {
		t.Fatalf("empresa B lê características da A: %d (%v)", len(cs), err)
	}
	if err := repo.DeactivatePlan(b.Ctx, plano.ID); err == nil {
		t.Fatal("empresa B desativou o plano da A")
	}
	code, err := repo.NextNCCode(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	nc, err := repo.CreateNC(a.Ctx, &entity.NonConformance{Code: code, ItemCode: item, NonConformQty: 1, Description: "risco", Severity: entity.SeverityMinor, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if abertas, err := repo.ListOpenNCs(b.Ctx); err != nil || len(abertas) != 0 {
		t.Fatalf("empresa B vê não conformidades da A: %d (%v)", len(abertas), err)
	}
	if _, err := repo.GetNCByID(b.Ctx, nc.ID); err == nil {
		t.Fatal("empresa B leu a não conformidade da A")
	}
	if err := repo.DispositionNC(b.Ctx, nc.ID, entity.DispositionScrap, actor.String()); err == nil {
		t.Fatal("empresa B deu destino à não conformidade da A")
	}
	if abertas, err := repo.ListOpenNCs(a.Ctx); err != nil || len(abertas) != 1 {
		t.Fatalf("empresa A não vê a própria não conformidade: %d (%v)", len(abertas), err)
	}
}
