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
	var status string
	var count int
	// A etapa exige inspeção e NÃO existe plano ativo. Isso não pode travar a
	// liberação: o plano é cadastro de outra tela e de outra pessoa, e recusar a
	// ordem parava a fábrica por uma lacuna que o operador não resolve. A ordem
	// sai liberada, com aviso, e sem registro de inspeção pendente inventado.
	if _, err := uc.ExecuteTransition(ctx, dto); err != nil {
		t.Fatalf("etapa marcada para inspeção sem plano ativo travou a liberação: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT status FROM planned_orders WHERE id=$1", order.ID).Scan(&status); err != nil || status != "RELEASED" {
		t.Fatalf("ordem não foi liberada: %s %v", status, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_orders WHERE planned_order_id=$1", order.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("OF não gerada: %d %v", count, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM quality_records r
		JOIN production_orders p ON p.id=r.production_order_id WHERE p.planned_order_id=$1`, order.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("conferência de qualidade criada sem plano ativo: %d %v", count, err)
	}

	// Volta ao estado planejado para exercitar a atomicidade a seguir. É reset de
	// cenário, feito em SQL de propósito: é justamente o caminho que o caso de uso
	// recusa (ordem com OF gerada não volta para planejada).
	testutil.Exec(t, pool, "DELETE FROM production_order_operations WHERE production_order_id IN (SELECT id FROM production_orders WHERE planned_order_id=$1)", order.ID)
	testutil.Exec(t, pool, "DELETE FROM production_orders WHERE planned_order_id=$1", order.ID)
	testutil.Exec(t, pool, "UPDATE planned_orders SET status='PLANNED' WHERE id=$1", order.ID)
	testutil.Exec(t, pool, "UPDATE route_operations SET inspection_required=false WHERE route_id=$1", route)

	// Lote com a mesma ordem duas vezes é recusado e não deixa nada atrás.
	dup := dto
	dup.OrderCodes = []int64{order.Code, order.Code}
	if _, err := uc.ExecuteTransition(ctx, dup); err == nil {
		t.Fatal("duplicate batch accepted")
	}
	if err := pool.QueryRow(ctx, "SELECT status FROM planned_orders WHERE id=$1", order.ID).Scan(&status); err != nil || status != "PLANNED" {
		t.Fatalf("state leaked: %s %v", status, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM production_orders WHERE planned_order_id=$1", order.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("OF leaked: %d %v", count, err)
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
	t.Log("Inspeção sem plano libera com aviso; lote duplicado não deixa rastro; 24 liberações simultâneas geraram exatamente uma OF e uma operação")
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
