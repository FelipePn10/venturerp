//go:build integration

package notification_test

import (
	"context"
	"fmt"
	"testing"

	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O mínimo vem da política e vale para o almoxarifado inteiro: dois endereços
// abaixo dele são um alerta só, e o saldo somado acima do mínimo não alerta.
func TestAlertaDeMinimoLeAPoliticaDoAlmoxarifado(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`,
			`DELETE FROM stock_replenishment_policies WHERE enterprise_id=$1`,
			`DELETE FROM stock_balances WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`,
			`DELETE FROM warehouse WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, a.ID)
		}
	})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	abaixo, acima := testutil.UniqueCode(), testutil.UniqueCode()
	for _, c := range []int64{abaixo, acima} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`, c, fmt.Sprint(c), almox, ator, a.ID)
		testutil.Exec(t, pool, `INSERT INTO stock_replenishment_policies(enterprise_id,item_code,warehouse_id,minimum_qty,source_type,created_by)
			VALUES($1,$2,$3,10,'COMPRA',$4)`, a.ID, c, almox, ator)
	}
	saldo := func(item int64, endereco string, q int) {
		testutil.Exec(t, pool, `INSERT INTO stock_balances(enterprise_id,item_code,warehouse_id,mask,address,quantity) VALUES($1,$2,$3,'',$4,$5)`, a.ID, item, almox, endereco, q)
	}
	saldo(abaixo, "A1", 3)
	saldo(abaixo, "A2", 4)
	saldo(acima, "A1", 6)
	saldo(acima, "A2", 6)

	if err := notificationrepo.New(pool).ScheduleOperationalAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	var total, doItem int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE aggregate_public_id=$2) FROM notification_outbox
		  WHERE enterprise_id=$1 AND event_key='ESTOQUE_ABAIXO_MINIMO'`, a.ID, fmt.Sprint(abaixo)).Scan(&total, &doItem); err != nil {
		t.Fatal(err)
	}
	if total != 1 || doItem != 1 {
		t.Fatalf("um alerta só, do item abaixo do mínimo: total %d, do item %d", total, doItem)
	}
}
