//go:build integration

package routing_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Roteiro desativado continua listável (com incluir inativos) e volta ao
// planejamento ao ser reativado. Se outro roteiro ativo já é o padrão do item,
// o reativado volta como alternativo — nunca dois padrões.
func TestIntegration_Routing_ReactivateRoute(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := routingrepo.New(q)
	ctx := testutil.TenantContext(t, pool)
	uid := uuid.New()
	itemCode := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, itemCode, uid)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE item_code = $1", itemCode)

	novo := func(alt int16, padrao bool) *entity.ManufacturingRoute {
		t.Helper()
		r, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), itemCode, nil, alt, ptrStr("roteiro"), padrao, nil, nil, uid)
		c, err := repo.CreateRoute(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	a := novo(1, true)
	if err := repo.DeactivateRoute(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if l, _ := repo.ListRoutesByItem(ctx, itemCode); len(l) != 0 {
		t.Fatalf("desativado ainda na lista de ativos: %d", len(l))
	}
	todos, err := repo.ListRoutesByItemWithInactive(ctx, itemCode)
	if err != nil || len(todos) != 1 || todos[0].IsActive {
		t.Fatalf("lista com inativos = %+v, %v", todos, err)
	}

	ok, err := repo.ReactivateRoute(ctx, a.ID)
	if err != nil || !ok {
		t.Fatalf("reativar: ok=%v err=%v", ok, err)
	}
	got, _ := repo.GetRouteByID(ctx, a.ID)
	if !got.IsActive || !got.IsStandard {
		t.Fatalf("reativado sem concorrente: ativo=%v padrão=%v, quer true/true", got.IsActive, got.IsStandard)
	}
	if ok, _ := repo.ReactivateRoute(ctx, a.ID); ok {
		t.Fatal("reativar um roteiro já ativo não pode afetar nada")
	}

	// Outro roteiro vira o padrão enquanto o primeiro está desativado.
	if err := repo.DeactivateRoute(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	novo(2, true)
	if ok, _ := repo.ReactivateRoute(ctx, a.ID); !ok {
		t.Fatal("reativar falhou")
	}
	got, _ = repo.GetRouteByID(ctx, a.ID)
	if !got.IsActive || got.IsStandard {
		t.Fatalf("reativado com outro padrão ativo: ativo=%v padrão=%v, quer true/false", got.IsActive, got.IsStandard)
	}

	// Outra empresa não reativa.
	if err := repo.DeactivateRoute(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	_, outra := testutil.DuasEmpresas(t, pool, nil)
	if ok, _ := repo.ReactivateRoute(outra.Ctx, a.ID); ok {
		t.Fatal("outra empresa reativou o roteiro")
	}
}
