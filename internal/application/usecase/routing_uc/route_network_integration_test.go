//go:build integration

package routing_uc

import (
	"context"
	"fmt"
	productionuc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"math"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestRouteAndNetworkRegisteredReferencesIntegration(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var enterprise int64
	if err := tx.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	q := sqlc.New(tx)
	repo := routingrepo.New(q)
	uc := NewRouteUseCase(repo, itemrepo.NewRepositoryItemSQLC(q))
	code := testutil.UniqueCode()
	business := fmt.Sprintf("BU-%d", code)
	user := uuid.New()
	if _, err := tx.Exec(ctx, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,$2,$1,$3,$4)", code, business, user, enterprise); err != nil {
		t.Fatal(err)
	}
	route, err := uc.Create(ctx, request.CreateRouteDTO{ItemCode: request.TextCode(business), Alternative: 1, IsStandard: true})
	if err != nil {
		t.Fatalf("aba 2, item cadastrado: %v", err)
	}
	operation, err := entity.NewOperation(testutil.UniqueCode(), "Montar", nil, entity.OriginInternal, nil, 1, 0, user)
	if err != nil {
		t.Fatal(err)
	}
	operation.RunTime = 90
	operation.SetupTime = 180
	operation.TimeUnit = "SEGUNDO"
	operation.RunBaseQty = 2
	op, err := repo.CreateOperation(ctx, operation)
	if err != nil {
		t.Fatal(err)
	}
	var steps []int64
	for i := 0; i < 3; i++ {
		step, err := entity.NewRouteOperation(route.ID, int16((i+1)*10), op.ID, nil, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		saved, err := repo.AddRouteOperation(ctx, step)
		if err != nil {
			t.Fatal(err)
		}
		steps = append(steps, saved.ID)
	}
	for _, edge := range []request.SetNetworkEdgeDTO{
		{RouteID: route.ID, PredecessorID: steps[0], SuccessorID: steps[2]},
		{RouteID: route.ID, PredecessorID: steps[1], SuccessorID: steps[2]},
	} {
		if _, err := uc.SetEdge(ctx, edge); err != nil {
			t.Fatalf("aba 5, etapas cadastradas: %v", err)
		}
	}
	if _, err := uc.SetEdge(ctx, request.SetNetworkEdgeDTO{RouteID: route.ID, PredecessorID: steps[2], SuccessorID: steps[0]}); err == nil {
		t.Fatal("ciclo aceito")
	}
	if _, err := uc.SetEdge(ctx, request.SetNetworkEdgeDTO{RouteID: route.ID, PredecessorID: steps[0], SuccessorID: 999999999}); err == nil {
		t.Fatal("etapa inexistente aceita")
	}
	var order int64
	if err := tx.QueryRow(ctx, "INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$1,10,'IN_PROGRESS',$2,$3) RETURNING id", code, user, enterprise).Scan(&order); err != nil {
		t.Fatal(err)
	}
	execution := productionuc.OrderOperationsUseCase{Q: q, Routing: repo, WithinTransaction: func(ctx context.Context, fn func(*sqlc.Queries) error) error {
		nested, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		defer nested.Rollback(ctx)
		if err := fn(sqlc.New(nested)); err != nil {
			return err
		}
		return nested.Commit(ctx)
	}}
	generated, err := execution.ExplodeRoute(ctx, order, 0)
	if err != nil {
		t.Fatalf("geração automática do roteiro da ordem: %v", err)
	}
	var ids []int64
	for _, step := range generated {
		ids = append(ids, step.ID)
		if math.Abs(step.PlannedHours-0.125) > 0.00001 || math.Abs(step.SetupHours-0.05) > 0.00001 {
			t.Fatalf("tempo do lote incorreto: %+v", step)
		}
	}
	again, err := execution.ExplodeRoute(ctx, order, route.ID)
	if err != nil || len(again) != 3 {
		t.Fatalf("geração repetida duplicou etapas: %d %v", len(again), err)
	}

	// Engineering edits after generation cannot rewrite the order graph.
	if _, err := tx.Exec(ctx, "DELETE FROM route_operation_network WHERE predecessor_id=ANY($1::bigint[])", steps); err != nil {
		t.Fatal(err)
	}

	for i, id := range ids {
		blocked, err := q.OperationPredecessorsPending(ctx, id)
		if err != nil || blocked != (i == 2) {
			t.Fatalf("paralelismo etapa %d bloqueada=%v err=%v", i, blocked, err)
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE production_order_operations SET status='DONE' WHERE id=$1", ids[0]); err != nil {
		t.Fatal(err)
	}
	if blocked, err := q.OperationPredecessorsPending(ctx, ids[2]); err != nil || !blocked {
		t.Fatal("junção liberada antes das duas predecessoras")
	}
	if _, err := tx.Exec(ctx, "UPDATE production_order_operations SET status='DONE' WHERE id=$1", ids[1]); err != nil {
		t.Fatal(err)
	}
	if blocked, err := q.OperationPredecessorsPending(ctx, ids[2]); err != nil || blocked {
		t.Fatal("junção não liberada")
	}
}
