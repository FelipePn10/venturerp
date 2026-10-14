//go:build integration

package routing_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Etapas, recursos e precedências do roteiro não têm coluna de empresa: valem
// pela empresa do roteiro. Outra empresa não inclui, altera, inativa nem lista
// nada pelo id, e a própria empresa não aponta centro de trabalho alheio.
func TestFilhosDoRoteiroIsoladosPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	uid := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM route_operation_network WHERE predecessor_id IN (SELECT ro.id FROM route_operations ro JOIN manufacturing_routes mr ON mr.id=ro.route_id WHERE mr.enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM route_operation_resources WHERE route_operation_id IN (SELECT ro.id FROM route_operations ro JOIN manufacturing_routes mr ON mr.id=ro.route_id WHERE mr.enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM route_operations WHERE route_id IN (SELECT id FROM manufacturing_routes WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM operations WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id = ANY($1)`, ids)
	})
	repo := routingrepo.New(sqlc.New(pool))
	centro := func(e testutil.Empresa) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO machine_types (code,name,type,requires_operator,created_by,enterprise_id) VALUES ($1,'CT','CUT',false,$2,$3) RETURNING id`,
			testutil.UniqueCode(), uid, e.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	wcA, wcB := centro(a), centro(b)
	op, _ := entity.NewOperation(testutil.UniqueCode(), "Corte", nil, entity.OriginInternal, nil, 1, 0, uuid.New())
	opA, err := repo.CreateOperation(a.Ctx, op)
	if err != nil {
		t.Fatal(err)
	}
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, uid)
	rt, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), item, nil, 1, nil, true, nil, nil, uid)
	rota, err := repo.CreateRoute(a.Ctx, rt)
	if err != nil {
		t.Fatal(err)
	}
	etapa := func(seq int16) *entity.RouteOperation {
		ro, _ := entity.NewRouteOperation(rota.ID, seq, opA.ID, nil, nil, nil, nil)
		return ro
	}
	e1, err := repo.AddRouteOperation(a.Ctx, etapa(10))
	if err != nil {
		t.Fatal(err)
	}
	e2, err := repo.AddRouteOperation(a.Ctx, etapa(20))
	if err != nil {
		t.Fatal(err)
	}

	recusa := func(nome string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: a empresa B (ou a referência alheia) passou", nome)
		}
	}
	_, err = repo.AddRouteOperation(b.Ctx, etapa(30))
	recusa("B inclui etapa no roteiro da A", err)
	outra := etapa(40)
	outra.WorkCenterID = &wcB
	_, err = repo.AddRouteOperation(a.Ctx, outra)
	recusa("A inclui etapa com centro da B", err)
	_, err = repo.UpdateRouteOperation(b.Ctx, &entity.RouteOperation{ID: e1.ID, Situation: entity.RouteOpInactive})
	recusa("B altera etapa da A", err)
	recusa("B inativa etapa da A", repo.RemoveRouteOperation(b.Ctx, e1.ID))

	_, err = repo.AddRouteOpResource(b.Ctx, &entity.RouteOpResource{RouteOperationID: e1.ID, WorkCenterID: wcB, Priority: 1, TimeFactor: 1})
	recusa("B inclui recurso na etapa da A", err)
	_, err = repo.AddRouteOpResource(a.Ctx, &entity.RouteOpResource{RouteOperationID: e1.ID, WorkCenterID: wcB, Priority: 1, TimeFactor: 1})
	recusa("A inclui recurso com centro da B", err)
	rec, err := repo.AddRouteOpResource(a.Ctx, &entity.RouteOpResource{RouteOperationID: e1.ID, WorkCenterID: wcA, Priority: 1, TimeFactor: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetRouteOpResource(b.Ctx, rec.ID)
	recusa("B lê recurso da A", err)
	_, err = repo.UpdateRouteOpResource(b.Ctx, &entity.RouteOpResource{ID: rec.ID, Priority: 9, TimeFactor: 2})
	recusa("B altera recurso da A", err)
	_, err = repo.SetRouteOpResourcePrimary(b.Ctx, rec.ID, e1.ID, wcA)
	recusa("B torna principal o recurso da A", err)
	recusa("B remove recurso da A", repo.RemoveRouteOpResource(b.Ctx, rec.ID))
	if l, err := repo.ListResourcesByRouteOp(b.Ctx, e1.ID); err != nil || len(l) != 0 {
		t.Fatalf("B lista recursos da etapa da A: %d (%v)", len(l), err)
	}
	if l, err := repo.ListResourcesByRoute(b.Ctx, rota.ID); err != nil || len(l) != 0 {
		t.Fatalf("B lista recursos do roteiro da A: %d (%v)", len(l), err)
	}

	_, err = repo.SetNetworkEdge(b.Ctx, &entity.NetworkEdge{PredecessorID: e1.ID, SuccessorID: e2.ID})
	recusa("B liga etapas da A", err)
	if _, err := repo.SetNetworkEdge(a.Ctx, &entity.NetworkEdge{PredecessorID: e1.ID, SuccessorID: e2.ID}); err != nil {
		t.Fatal(err)
	}
	recusa("B apaga precedência da A", repo.DeleteNetworkEdge(b.Ctx, e1.ID, e2.ID))

	// Nada da A mudou; e a própria A continua operando.
	var ativo bool
	var prioridade int
	if err := pool.QueryRow(base, `SELECT ro.is_active, r.priority FROM route_operations ro JOIN route_operation_resources r ON r.route_operation_id = ro.id WHERE ro.id = $1`, e1.ID).Scan(&ativo, &prioridade); err != nil || !ativo || prioridade != 1 {
		t.Fatalf("a etapa/recurso da A mudou: ativo=%v prioridade=%d (%v)", ativo, prioridade, err)
	}
	if l, err := repo.ListResourcesByRouteOp(a.Ctx, e1.ID); err != nil || len(l) != 1 {
		t.Fatalf("A lista os próprios recursos: %d (%v)", len(l), err)
	}
	if err := repo.DeleteNetworkEdge(a.Ctx, e1.ID, e2.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveRouteOpResource(a.Ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveRouteOperation(a.Ctx, e2.ID); err != nil {
		t.Fatal(err)
	}
}

// Trocar o recurso principal é tudo ou nada: se falhar depois de limpar o
// principal atual, a etapa continua com ele.
func TestTrocaDeRecursoPrincipalAtomica(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	uid := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM route_operation_resources WHERE route_operation_id IN (SELECT ro.id FROM route_operations ro JOIN manufacturing_routes mr ON mr.id=ro.route_id WHERE mr.enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM route_operations WHERE route_id IN (SELECT id FROM manufacturing_routes WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM operations WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id = ANY($1)`, ids)
	})
	repo := routingrepo.New(sqlc.New(pool))
	var wc int64
	if err := pool.QueryRow(base, `INSERT INTO machine_types (code,name,type,requires_operator,created_by,enterprise_id) VALUES ($1,'CT','CUT',false,$2,$3) RETURNING id`,
		testutil.UniqueCode(), uid, a.ID).Scan(&wc); err != nil {
		t.Fatal(err)
	}
	op, _ := entity.NewOperation(testutil.UniqueCode(), "Corte", nil, entity.OriginInternal, nil, 1, 0, uuid.New())
	opA, err := repo.CreateOperation(a.Ctx, op)
	if err != nil {
		t.Fatal(err)
	}
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, uid)
	rt, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), item, nil, 1, nil, true, nil, nil, uid)
	rota, err := repo.CreateRoute(a.Ctx, rt)
	if err != nil {
		t.Fatal(err)
	}
	etapa := func(seq int16) int64 {
		ro, _ := entity.NewRouteOperation(rota.ID, seq, opA.ID, nil, nil, nil, nil)
		e, err := repo.AddRouteOperation(a.Ctx, ro)
		if err != nil {
			t.Fatal(err)
		}
		return e.ID
	}
	e1, e2 := etapa(10), etapa(20)
	recurso := func(op int64) int64 {
		r, err := repo.AddRouteOpResource(a.Ctx, &entity.RouteOpResource{RouteOperationID: op, WorkCenterID: wc, Priority: 1, TimeFactor: 1})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	r1, r3 := recurso(e1), recurso(e2)
	if _, err := repo.SetRouteOpResourcePrimary(a.Ctx, r1, e1, wc); err != nil {
		t.Fatal(err)
	}
	// Recurso de outra etapa: a limpeza da etapa 1 já rodou quando a
	// inconsistência aparece — e tem de ser desfeita.
	if _, err := repo.SetRouteOpResourcePrimary(a.Ctx, r3, e1, wc); err == nil {
		t.Fatal("aceitou recurso de outra etapa como principal")
	}
	var principal bool
	if err := pool.QueryRow(base, `SELECT is_primary FROM route_operation_resources WHERE id=$1`, r1).Scan(&principal); err != nil || !principal {
		t.Fatalf("a falha no meio deixou a etapa sem o recurso principal (%v)", err)
	}
	if err := pool.QueryRow(base, `SELECT is_primary FROM route_operation_resources WHERE id=$1`, r3).Scan(&principal); err != nil || principal {
		t.Fatalf("o recurso da outra etapa ficou principal (%v)", err)
	}
}
