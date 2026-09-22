//go:build integration

package production_order_test

import (
	"context"
	"crypto/sha256"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	entity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"testing"
)

func TestFinalDeliveryRequiresCompletedRoutingIntegration(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	var enterprise int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	code := testutil.UniqueCode()
	uid := uuid.New()
	var order, op int64
	if _, err := pool.Exec(ctx, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$1,$2,$3)", code, uid, enterprise); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", code)
	if err := pool.QueryRow(ctx, "INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$1,10,'IN_PROGRESS',$2,$3) RETURNING id", code, uid, enterprise).Scan(&order); err != nil {
		t.Fatal(err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id=$1", order)
	if err := pool.QueryRow(ctx, "INSERT INTO production_order_operations(production_order_id,sequence,operation_name,enterprise_id) VALUES($1,10,'Montagem',$2) RETURNING id", order, enterprise).Scan(&op); err != nil {
		t.Fatal(err)
	}
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	delivery := &entity.ProductionDelivery{ProductionOrderID: order, Quantity: decimal.Zero, IdempotencyKey: uuid.NewString(), MovementClass: "EPP", WarehouseID: code, IsFinal: true, CreatedBy: uid}
	if _, err := repo.RegisterDeliveryWithMovements(ctx, delivery, nil); err == nil || !strings.Contains(err.Error(), "etapas do roteiro") {
		t.Fatalf("entrega não bloqueada pelo roteiro: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM production_deliveries WHERE production_order_id=$1", order).Scan(&count); err != nil || count != 0 {
		t.Fatalf("entrega parcial persistida: %d %v", count, err)
	}
	for _, status := range []string{"PAUSED", "INTERRUPTED"} {
		if _, err := pool.Exec(ctx, "UPDATE production_order_operations SET status=$1 WHERE id=$2", status, op); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.RegisterDeliveryWithMovements(ctx, delivery, nil); err == nil {
			t.Fatalf("entrega aceita com etapa %s", status)
		}
	}
	// Scanner follows dependencies and counts finished output only on the final operation.
	var user uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT id FROM users LIMIT 1").Scan(&user); err != nil {
		t.Fatal(err)
	}
	var second int64
	if err := pool.QueryRow(ctx, "INSERT INTO production_order_operations(production_order_id,sequence,operation_name,enterprise_id) VALUES($1,20,'Acabamento',$2) RETURNING id", order, enterprise).Scan(&second); err != nil {
		t.Fatal(err)
	}
	device := "routing-test-" + uuid.NewString()
	defer testutil.Exec(t, pool, "DELETE FROM production_appointments WHERE production_order_id=$1", order)
	defer testutil.Exec(t, pool, "DELETE FROM production_scan_events WHERE device_id=$1", device)
	tokens := map[int64][]byte{}
	for _, id := range []int64{op, second} {
		hash := sha256.Sum256([]byte(uuid.NewString()))
		tokens[id] = hash[:]
		if err := repo.CreateScanToken(ctx, &entity.ScanToken{EnterpriseID: enterprise, ProductionOrderID: order, OperationID: &id, TokenHash: hash[:], CreatedBy: user}); err != nil {
			t.Fatal(err)
		}
	}
	command := func(id int64) entity.ScanCommand {
		return entity.ScanCommand{EnterpriseID: enterprise, UserID: user, TokenHash: tokens[id], Action: entity.ScanAppoint, IdempotencyKey: uuid.NewString(), DeviceID: device, Fingerprint: []byte{1}, GoodQuantity: decimal.NewFromInt(2), Hours: decimal.NewFromInt(1), CompleteOperation: true}
	}
	if _, err := repo.ExecuteScan(ctx, command(second)); err == nil {
		t.Fatal("scanner ignorou predecessora")
	}
	if _, err := repo.ExecuteScan(ctx, command(op)); err == nil {
		t.Fatal("scanner aceitou etapa interrompida")
	}
	if _, err := pool.Exec(ctx, "UPDATE production_order_operations SET status='PENDING' WHERE id=$1", op); err != nil {
		t.Fatal(err)
	}
	firstCommand := command(op)
	if _, err := repo.ExecuteScan(ctx, firstCommand); err != nil {
		t.Fatal(err)
	}
	var produced float64
	if err := pool.QueryRow(ctx, "SELECT produced_qty FROM production_orders WHERE id=$1", order).Scan(&produced); err != nil || produced != 0 {
		t.Fatalf("etapa intermediária contou acabado: %v %v", produced, err)
	}
	if replay, err := repo.ExecuteScan(ctx, firstCommand); err != nil || !replay.Replayed {
		t.Fatalf("idempotência do scanner: %v", err)
	}
	if _, err := repo.ExecuteScan(ctx, command(second)); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT produced_qty FROM production_orders WHERE id=$1", order).Scan(&produced); err != nil || produced != 2 {
		t.Fatalf("produção final incorreta: %v %v", produced, err)
	}
	finish := command(second)
	finish.Action = entity.ScanComplete
	result, err := repo.ExecuteScan(ctx, finish)
	if err != nil || result.NextAction != "DELIVER_PRODUCTION" || result.Status != "EM_ANDAMENTO" {
		t.Fatalf("scanner não direcionou para entrega: %+v %v", result, err)
	}

	// Competing final deliveries serialize: only one can close the order.
	defer testutil.Exec(t, pool, "DELETE FROM production_deliveries WHERE production_order_id=$1", order)
	results := make(chan error, 24)
	start := make(chan struct{})
	for i := 0; i < 24; i++ {
		go func() {
			<-start
			d := *delivery
			d.IdempotencyKey = uuid.NewString()
			_, err := repo.RegisterDeliveryWithMovements(ctx, &d, nil)
			results <- err
		}()
	}
	close(start)
	accepted := 0
	for i := 0; i < 24; i++ {
		if <-results == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("concurrent final deliveries accepted=%d", accepted)
	}
	// A second request must never reopen a completed order.
	delivery.IsFinal = false
	if _, err := repo.RegisterDeliveryWithMovements(ctx, delivery, nil); err == nil {
		t.Fatal("delivery reopened a completed order")
	}
}
