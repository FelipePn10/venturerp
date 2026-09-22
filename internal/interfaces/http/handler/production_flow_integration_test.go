//go:build integration

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	entity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	prodrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	structurerepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Real HTTP handlers + use cases + PostgreSQL. Only identity is provided by the
// isolated test server; stock, BOM, route and execution are not mocked.
func TestProductionHTTPFlowAtomicCreationAndStock(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	enterprise := testutil.EnterpriseID(t, ctx)
	actor := testutil.Actor(t, pool)
	q := sqlc.New(pool)
	repo := prodrepo.NewProductionOrderRepositoryPGX(pool)
	routing := routingrepo.New(q)
	authorization := &auth.AuthService{}
	finished, component := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, finished, actor)
	testutil.SeedItem(t, pool, ctx, component, actor)
	var warehouse, route, operation int64
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,enterprise_id,disposition,reservations_allowed) VALUES($1,'HTTP test',$2,'INTERNO','NORMAL',$3,true,true) RETURNING id`, fmt.Sprint(finished), actor, enterprise).Scan(&warehouse); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE items SET warehouse_code=$1,warehouse_automatic_low=true WHERE code=$2", warehouse, component)
	exec("INSERT INTO stock_balances(enterprise_id,item_code,mask,warehouse_id,quantity) VALUES($1,$2,'',$3,30)", enterprise, component, warehouse)
	exec("INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES($1,$2,3,10,$3)", finished, component, actor)
	if err := pool.QueryRow(ctx, `INSERT INTO manufacturing_routes(code,item_code,is_standard,created_by,enterprise_id) VALUES($1,$1,true,$2,$3) RETURNING id`, finished, actor, enterprise).Scan(&route); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO operations(code,name,created_by,enterprise_id) VALUES($1,'Montagem HTTP',$2,$3) RETURNING id`, finished, actor, enterprise).Scan(&operation); err != nil {
		t.Fatal(err)
	}
	exec("INSERT INTO route_operations(route_id,sequence,operation_id,inspection_required) VALUES($1,10,$2,true)", route, operation)
	exec(`INSERT INTO inspection_plans(item_code,route_operation_id,point_type,description,created_by) SELECT $1,id,'PROCESSO','Inspeção HTTP',$3 FROM route_operations WHERE route_id=$2`, finished, route, actor)
	t.Cleanup(func() {
		for _, query := range []string{
			"DELETE FROM stock_movements WHERE enterprise_id=$1 AND item_code=ANY($2::bigint[])",
			"DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code=ANY($2::bigint[])",
			"DELETE FROM quality_records WHERE production_order_id IN(SELECT id FROM production_orders WHERE enterprise_id=$1 AND item_code=ANY($2::bigint[]))",
			"DELETE FROM inspection_plans WHERE item_code=ANY($2::bigint[]) AND $1::bigint>0",
			"DELETE FROM production_deliveries WHERE production_order_id IN(SELECT id FROM production_orders WHERE enterprise_id=$1 AND item_code=ANY($2::bigint[]))",
			"DELETE FROM production_orders WHERE enterprise_id=$1 AND item_code=ANY($2::bigint[])",
			"DELETE FROM manufacturing_routes WHERE enterprise_id=$1 AND item_code=ANY($2::bigint[])",
		} {
			testutil.Exec(t, pool, query, enterprise, []int64{finished, component})
		}
		testutil.Exec(t, pool, "DELETE FROM operations WHERE id=$1", operation)
		testutil.Exec(t, pool, "DELETE FROM item_structures WHERE parent_code=$1", finished)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=ANY($1::bigint[])", []int64{finished, component})
		testutil.Exec(t, pool, "DELETE FROM warehouse WHERE id=$1", warehouse)
	})
	ops := &uc.OrderOperationsUseCase{Q: q, Auth: authorization, Routing: routing, WithinTransaction: func(ctx context.Context, fn func(*sqlc.Queries) error) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if err := fn(q.WithTx(tx)); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}}
	structure := structurerepo.NewItemStructureRepository(q)
	create := &uc.CreateProductionOrderUseCase{Repo: repo, Auth: authorization, Items: itemrepo.NewRepositoryItemSQLC(q), Structure: structure, Routing: routing, OrderOps: ops}
	create.CreateAtomically = func(ctx context.Context, order *entity.ProductionOrder, materials []*entity.ProductionOrderMaterial, routeID int64) (*entity.ProductionOrder, error) {
		return repo.CreateWithMaterialsAndCallback(ctx, order, materials, func(ctx context.Context, tx pgx.Tx, created *entity.ProductionOrder) error {
			inner := *ops
			inner.Q = q.WithTx(tx)
			inner.WithinTransaction = nil
			if _, err := inner.ExplodeRoute(ctx, created.ID, routeID); err != nil {
				return err
			}
			if order.Notes != nil && *order.Notes == "inject failure" {
				return fmt.Errorf("injected failure after steps")
			}
			return nil
		})
	}
	h := &ProductionOrderHandler{createUC: create, startUC: &uc.StartProductionOrderUseCase{Repo: repo, Auth: authorization}, orderOpsUC: ops, completeUC: &uc.CompleteProductionOrderUseCase{Repo: repo, Auth: authorization, Structure: structure, StockRepo: stockrepo.NewStockRepositorySQLC(pool)}}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := *ctx.Value(contextkey.UserKey).(*security.AuthUser)
			if r.Header.Get("X-Test-Foreign") == "true" {
				u.EnterpriseID += 999999
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextkey.UserKey, &u)))
		})
	})
	router.Post("/orders", h.Create)
	router.Post("/orders/{id}/start", h.Start)
	router.Get("/orders/{id}/operations", h.ListOrderOperations)
	router.Post("/advance", h.AdvanceOperation)
	router.Post("/orders/{id}/complete", h.Complete)
	server := httptest.NewServer(router)
	defer server.Close()
	call := func(method, path, body string, foreign bool) (int, []byte) {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if foreign {
			req.Header.Set("X-Test-Foreign", "true")
		}
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, data
	}
	payload := fmt.Sprintf(`{"item_code":"%d","planned_qty":2,"warehouse_id":%d,"notes":"inject failure"}`, finished, warehouse)
	if status, body := call("POST", "/orders", payload, false); status < 400 {
		t.Fatalf("injected failure accepted: %d %s", status, body)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_orders WHERE item_code=$1", finished).Scan(&count); err != nil || count != 0 {
		t.Fatalf("creation leaked: %d %v", count, err)
	}
	payload = strings.ReplaceAll(payload, "inject failure", "production test")
	status, body := call("POST", "/orders", payload, false)
	if status != 201 {
		t.Fatalf("create: %d %s", status, body)
	}
	var order struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &order); err != nil || order.ID == 0 {
		t.Fatalf("created order: %s %v", body, err)
	}
	path := fmt.Sprintf("/orders/%d", order.ID)
	status, body = call("POST", path+"/start", "{}", false)
	if status != 200 {
		t.Fatalf("start: %d %s", status, body)
	}
	if status, _ := call("GET", path+"/operations", "", true); status < 400 {
		t.Fatal("cross-tenant operations exposed")
	}
	_, body = call("GET", path+"/operations", "", false)
	var steps []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &steps); err != nil || len(steps) != 1 {
		t.Fatalf("steps: %s %v", body, err)
	}
	delivery := fmt.Sprintf(`{"quantity":2,"final":true,"warehouse_id":%d,"idempotency_key":"%s"}`, warehouse, uuid.NewString())
	if status, _ := call("POST", path+"/complete", delivery, false); status < 400 {
		t.Fatal("unfinished routing delivered")
	}
	for _, state := range []string{"IN_PROGRESS", "PAUSED", "IN_PROGRESS", "INTERRUPTED", "IN_PROGRESS", "DONE"} {
		status, body = call("POST", "/advance", fmt.Sprintf(`{"operation_id":%d,"status":"%s","reason":"HTTP test"}`, steps[0].ID, state), false)
		if status != 200 {
			t.Fatalf("%s: %d %s", state, status, body)
		}
	}

	if status, body := call("POST", path+"/complete", delivery, false); status != 422 {
		t.Fatalf("quality gate: %d %s", status, body)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM quality_records WHERE production_order_id=$1 AND result='PENDENTE' AND created_by=$2", order.ID, actor).Scan(&count); err != nil || count != 1 {
		t.Fatalf("inspection not created with actor: %d %v", count, err)
	}
	exec("UPDATE quality_records SET result='APROVADO',inspected_qty=2,approved_qty=2 WHERE production_order_id=$1", order.ID)
	for i := 0; i < 2; i++ {
		status, body = call("POST", path+"/complete", delivery, false)
		if status != 200 {
			t.Fatalf("delivery/replay: %d %s", status, body)
		}
	}

	for _, changed := range []string{
		strings.Replace(delivery, `"quantity":2`, `"quantity":3`, 1),
		strings.Replace(delivery, `"final":true`, `"final":false`, 1),
	} {
		if status, body := call("POST", path+"/complete", changed, false); status != 422 {
			t.Fatalf("changed replay accepted: %d %s", status, body)
		}
	}

	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_operation_execution_events WHERE operation_id=$1 AND actor_id=$2 AND source='OPERATION'", steps[0].ID, actor).Scan(&count); err != nil || count != 6 {
		t.Fatalf("audit actor/history count=%d err=%v", count, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE production_operation_execution_events SET new_status='PENDING' WHERE operation_id=$1", steps[0].ID); err == nil {
		t.Fatal("execution history can be rewritten")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM production_operation_execution_events WHERE operation_id=$1", steps[0].ID); err == nil {
		t.Fatal("execution history can be deleted")
	}
	status, body = call("GET", path+"/operations", "", false)
	if status != 200 || !strings.Contains(string(body), "execution_history") {
		t.Fatalf("history missing from API: %d %s", status, body)
	}
	var componentBalance, finishedBalance float64
	if err := pool.QueryRow(ctx, "SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 AND mask=''", enterprise, component, warehouse).Scan(&componentBalance); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 AND mask=''", enterprise, finished, warehouse).Scan(&finishedBalance); err != nil {
		t.Fatal(err)
	}
	if componentBalance != 24 || finishedBalance != 2 {
		t.Fatalf("stock: component=%v finished=%v", componentBalance, finishedBalance)
	}
	t.Log("HTTP: rollback da criação, isolamento, execução, entrega/replay e estoques 30-6=24 / 0+2=2 aprovados")
}
