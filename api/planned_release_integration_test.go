//go:build integration

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	productionuc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	prodrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	structurerepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestAtomicPlannedReleaseRollbackAndConcurrentRetry(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	enterprise := testutil.EnterpriseID(t, ctx)
	actor := testutil.Actor(t, pool)
	q := sqlc.New(pool)
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, item, actor)
	var warehouse, route, operation int64
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,enterprise_id,disposition,reservations_allowed) VALUES($1,'Atomic release',$2,'INTERNO','NORMAL',$3,true,true) RETURNING id`, fmt.Sprint(item), actor, enterprise).Scan(&warehouse); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO manufacturing_routes(code,item_code,is_standard,created_by,enterprise_id) VALUES($1,$1,true,$2,$3) RETURNING id`, item, actor, enterprise).Scan(&route); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO operations(code,name,created_by,enterprise_id) VALUES($1,'Atomic release',$2,$3) RETURNING id`, item, actor, enterprise).Scan(&operation); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO route_operations(route_id,sequence,operation_id,inspection_required) VALUES($1,10,$2,true)`, route, operation)
	planned := plannedrepo.NewPlannedOrderRepositorySQLC(q)
	order, err := planned.Create(ctx, &plannedentity.PlannedOrder{OrderNumber: testutil.UniqueCode(), ItemCode: item, Quantity: 2, QuantityCorrected: 2, OrderType: types.OrderProduction, Status: types.StatusPlanned, DemandType: types.DemandType("INDEPENDENT"), NeedDate: time.Now(), WarehouseCode: &warehouse, IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM production_orders WHERE enterprise_id=$1 AND item_code=$2", enterprise, item)
		testutil.Exec(t, pool, "DELETE FROM planned_orders WHERE id=$1", order.ID)
		testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE id=$1", route)
		testutil.Exec(t, pool, "DELETE FROM operations WHERE id=$1", operation)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", item)
		testutil.Exec(t, pool, "DELETE FROM warehouse WHERE id=$1", warehouse)
	})
	authorization := &auth.AuthService{}
	uc := &planned_order_uc.FirmPlannedOrderUseCase{Auth: authorization, Structure: structurerepo.NewItemStructureRepository(q), Items: itemrepo.NewRepositoryItemSQLC(q)}
	wireAtomicPlannedRelease(pool, q, uc, &productionuc.OrderOperationsUseCase{Auth: authorization, Q: q})
	dto := request.TransitionPlannedOrderDTO{OrderCodes: []int64{order.Code}, Target: "RELEASED"}
	// Inspection is required, but its plan is absent: failure occurs after OF and
	// operation insertion, so neither those rows nor RELEASED may survive.
	if _, err := uc.ExecuteTransition(ctx, dto); err == nil {
		t.Fatal("missing inspection plan accepted")
	}
	var status string
	var count int
	if err := pool.QueryRow(ctx, "SELECT status FROM planned_orders WHERE id=$1", order.ID).Scan(&status); err != nil || status != "PLANNED" {
		t.Fatalf("state leaked: %s %v", status, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_orders WHERE planned_order_id=$1", order.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("OF leaked: %d %v", count, err)
	}
	testutil.Exec(t, pool, "UPDATE route_operations SET inspection_required=false WHERE route_id=$1", route)
	dup := dto
	dup.OrderCodes = []int64{order.Code, order.Code}
	if _, err := uc.ExecuteTransition(ctx, dup); err == nil {
		t.Fatal("duplicate batch accepted")
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.ExecuteTransition(runCtx, dto)
			if err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_orders WHERE planned_order_id=$1", order.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate/missing OF: %d %v", count, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_order_operations op JOIN production_orders p ON p.id=op.production_order_id WHERE p.planned_order_id=$1", order.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate/missing operation: %d %v", count, err)
	}
	dto.Target = "PLANNED"
	if _, err := uc.ExecuteTransition(ctx, dto); err == nil {
		t.Fatal("generated OF returned to planned, allowing duplicate release")
	}
	t.Log("Failed release rolled back; 24 simultaneous retries created exactly one OF and one operation")
}

func TestProductionOrderNumbersConcurrent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, cancel := context.WithTimeout(testutil.TenantContext(t, pool), 30*time.Second)
	defer cancel()
	repo := prodrepo.NewProductionOrderRepositoryPGX(pool)
	nums := make(chan int64, 24)
	failures := make(chan error, 24)
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := repo.GetNextOrderNumber(ctx)
			if err != nil {
				failures <- err
				return
			}
			nums <- n
		}()
	}
	wg.Wait()
	close(nums)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	seen := map[int64]bool{}
	for n := range nums {
		if seen[n] {
			t.Errorf("number %d reserved twice", n)
		}
		seen[n] = true
	}
	if len(seen) != 24 {
		t.Fatalf("unique numbers=%d, expected 24", len(seen))
	}
}
