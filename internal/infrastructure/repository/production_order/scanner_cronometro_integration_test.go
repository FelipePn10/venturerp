//go:build integration

package production_order_test

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TestLeituraConcluiPeloCronometro: concluir a etapa pelo código de barras
// sem horas grava o tempo corrido (40 min → ~0,67 h); a etapa esquecida em
// andamento (15 h) é recusada até alguém informar as horas.
func TestLeituraConcluiPeloCronometro(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	var enterprise int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	user := testutil.Actor(t, pool)
	code := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$1,$2,$3)", code, user, enterprise)
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM items WHERE code=$1", code) })
	var order int64
	if err := pool.QueryRow(ctx, "INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$1,10,'IN_PROGRESS',$2,$3) RETURNING id", code, user, enterprise).Scan(&order); err != nil {
		t.Fatal(err)
	}
	device := "crono-" + uuid.NewString()
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM production_scan_events WHERE device_id=$1", device)
		testutil.Exec(t, pool, "DELETE FROM production_appointments WHERE production_order_id=$1", order)
		testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id=$1", order)
	})
	etapa := func(seq int, desde string) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO production_order_operations(production_order_id,sequence,operation_name,status,started_at,enterprise_id)
			VALUES($1,$2,'Etapa lida',$4,clock_timestamp() - $3::interval,$5) RETURNING id`, order, seq, desde, "IN_PROGRESS", enterprise).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	normal, esquecida := etapa(10, "40 minutes"), etapa(20, "15 hours")
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	tokens := map[int64][]byte{}
	for _, id := range []int64{normal, esquecida} {
		h := sha256.Sum256([]byte(uuid.NewString()))
		tokens[id] = h[:]
		if err := repo.CreateScanToken(ctx, &entity.ScanToken{EnterpriseID: enterprise, ProductionOrderID: order, OperationID: &id, TokenHash: h[:], CreatedBy: user}); err != nil {
			t.Fatal(err)
		}
	}
	ler := func(id int64, horas int64) error {
		_, err := repo.ExecuteScan(ctx, entity.ScanCommand{EnterpriseID: enterprise, UserID: user, TokenHash: tokens[id], Action: entity.ScanAppoint,
			IdempotencyKey: uuid.NewString(), DeviceID: device, Fingerprint: []byte{1}, GoodQuantity: decimal.NewFromInt(2), Hours: decimal.NewFromInt(horas), CompleteOperation: true})
		return err
	}
	horas := func(id int64) float64 {
		var h float64
		if err := pool.QueryRow(ctx, "SELECT COALESCE(actual_hours,0)::float8 FROM production_order_operations WHERE id=$1", id).Scan(&h); err != nil {
			t.Fatal(err)
		}
		return h
	}
	if err := ler(normal, 0); err != nil {
		t.Fatal(err)
	}
	if h := horas(normal); h < 0.66 || h > 0.75 {
		t.Fatalf("leitura sem horas deveria gravar ~0,67 h pelo cronômetro; gravou %v", h)
	}
	if err := ler(esquecida, 0); err == nil || !strings.Contains(err.Error(), "ficou aberta") {
		t.Fatalf("etapa aberta há 15 h deveria pedir as horas: %v", err)
	}
	if h := horas(esquecida); h != 0 {
		t.Fatalf("a recusa não pode gravar horas: %v", h)
	}
	if err := ler(esquecida, 8); err != nil {
		t.Fatalf("com as horas informadas a leitura conclui: %v", err)
	}
	if h := horas(esquecida); h != 8 {
		t.Fatalf("horas informadas: %v", h)
	}
}
