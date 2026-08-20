//go:build integration

package notification

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestScheduleOperationalAlertsUsesCurrentWarehouseSchema(t *testing.T) {
	pool := testutil.Pool(t)
	if err := New(pool).ScheduleOperationalAlerts(context.Background()); err != nil {
		t.Fatalf("avaliar alertas operacionais no PostgreSQL: %v", err)
	}
}
