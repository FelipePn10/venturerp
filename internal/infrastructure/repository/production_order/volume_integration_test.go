//go:build integration

package production_order_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestManufacturingIndustrialVolumeMaintenanceConsultation(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	rowsCount := 2000
	if raw := os.Getenv("MANUFACTURING_VOLUME_ROWS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 100 {
			rowsCount = parsed
		}
	}
	base := int64(100_000_000 + time.Now().UnixNano()%500_000_000)
	_, err := pool.Exec(ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,origin_type,created_by,enterprise_id) SELECT $1+g,$2+g,100,'RELEASED','MANUAL',$3,$4 FROM generate_series(1,$5) g`, base, base, uuid.New(), enterpriseID, rowsCount)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM production_orders WHERE enterprise_id=$1 AND order_number>$2 AND order_number<=$2+$3", enterpriseID, base, rowsCount)
	})
	started := time.Now()
	views, err := productionrepo.NewProductionOrderRepositoryPGX(pool).GetMaintenance(ctx, nil)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, view := range views {
		if view.ProductionOrder.OrderNumber > base && view.ProductionOrder.OrderNumber <= base+int64(rowsCount) {
			matched++
		}
	}
	if matched != rowsCount {
		t.Fatalf("matched=%d want=%d", matched, rowsCount)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("industrial volume consultation took %s for %d rows (limit 5s)", elapsed, rowsCount)
	}
	t.Logf("industrial-volume rows=%d elapsed=%s", rowsCount, elapsed)
}
