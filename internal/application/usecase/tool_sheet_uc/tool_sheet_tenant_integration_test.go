//go:build integration

package tool_sheet_uc_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/tool_sheet_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestToolSheetIsIsolatedByTenant: a ficha pegava "a primeira empresa do
// banco" e listava as ordens de produção de todas as empresas.
func TestToolSheetIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	item := int64(1_720_000_000 + testutil.UniqueCode()%10_000_000)
	var ordem, etapa int64
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM production_order_operations WHERE production_order_id=$1`, ordem)
		_, _ = pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, ordem)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code=$1`, item)
	})
	actor := testutil.Actor(t, pool)
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,created_by,enterprise_id) VALUES($1,$1,1,$2,$3) RETURNING id`,
		item, actor, a.ID).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_order_operations(enterprise_id,production_order_id,sequence,operation_name) VALUES($1,$2,10,'CORTE') RETURNING id`,
		a.ID, ordem).Scan(&etapa); err != nil {
		t.Fatal(err)
	}
	uc := tool_sheet_uc.New(sqlc.New(pool))
	busca := strconv.FormatInt(item, 10)

	if l, err := uc.ListOrders(b.Ctx, busca); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d ordem(ns) da A (%v)", len(l), err)
	}
	if _, err := uc.GetSheet(b.Ctx, ordem); err == nil {
		t.Fatal("empresa B abriu a ficha da ordem da A")
	}
	if _, err := uc.ListSubstitutions(b.Ctx, etapa, 0); err == nil {
		t.Fatal("empresa B leu substituições da etapa da A")
	}
	l, err := uc.ListOrders(a.Ctx, busca)
	if err != nil || len(l) != 1 || l[0].EnterpriseID != a.ID {
		t.Fatalf("empresa A deveria ver a própria ordem rotulada com ela: %+v %v", l, err)
	}
	if _, err := uc.GetSheet(a.Ctx, ordem); err != nil {
		t.Fatalf("empresa A não abriu a própria ficha: %v", err)
	}
	if _, err := uc.ListSubstitutions(a.Ctx, etapa, 0); err != nil {
		t.Fatalf("empresa A não leu as substituições da própria etapa: %v", err)
	}
}
