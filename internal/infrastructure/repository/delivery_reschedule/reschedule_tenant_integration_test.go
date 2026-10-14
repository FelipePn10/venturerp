//go:build integration

package delivery_reschedule_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	reschedulerepo "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestDeliveryRescheduleIsIsolatedByTenant: a reprogramação comparava o
// enterprise_code do pedido com tenant.ID — na empresa 1 os dois valem 1 e o
// erro não aparecia; na segunda empresa a prévia não achava o pedido. As
// consultas por código não filtravam empresa nenhuma.
func TestDeliveryRescheduleIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	var pedido int64
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM delivery_reschedules WHERE sales_order_code=$1`, pedido)
		_, _ = pool.Exec(base, `DELETE FROM delivery_reschedule_batches WHERE sales_order_code=$1`, pedido)
		_, _ = pool.Exec(base, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, pedido)
		_, _ = pool.Exec(base, `DELETE FROM sales_orders WHERE code=$1`, pedido)
	})
	actor := testutil.Actor(t, pool)
	item := int64(1_700_000_000 + testutil.UniqueCode()%10_000_000)
	entrega := time.Date(2030, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := pool.QueryRow(base, `INSERT INTO sales_orders(order_number,enterprise_code,created_by) VALUES($1,$2,$3) RETURNING code`, item, a.Code, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	var linha int64
	if err := pool.QueryRow(base, `INSERT INTO sales_order_items(sales_order_code,item_code,requested_qty,delivery_date) VALUES($1,$2,10,$3::date) RETURNING code`, pedido, item, entrega).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	repo := delivery_reschedule.NewDeliveryRescheduleRepositorySQLC(sqlc.New(pool), pool)

	prev, err := repo.Preview(a.Ctx, pedido)
	if err != nil {
		t.Fatalf("prévia na empresa do pedido: %v", err)
	}
	if len(prev) != 1 || prev[0].OpenQty != "10.0000" || !prev[0].CanReschedule {
		t.Fatalf("prévia inesperada: %+v", prev)
	}
	if _, err := repo.Preview(b.Ctx, pedido); !errors.Is(err, reschedulerepo.ErrOrderNotFound) {
		t.Fatalf("empresa B não deveria achar o pedido da A: %v", err)
	}

	// Linha totalmente atendida não pode ser reprogramada.
	if _, err := pool.Exec(base, `UPDATE sales_order_items SET attended_qty=10 WHERE code=$1`, linha); err != nil {
		t.Fatal(err)
	}
	if prev, err := repo.Preview(a.Ctx, pedido); err != nil || prev[0].CanReschedule {
		t.Fatalf("linha atendida apareceu como reprogramável: %+v %v", prev, err)
	}
	if _, err := pool.Exec(base, `UPDATE sales_order_items SET attended_qty=0 WHERE code=$1`, linha); err != nil {
		t.Fatal(err)
	}

	nova := entrega.AddDate(0, 0, 7)
	cmd := reschedulerepo.BatchCommand{ID: uuid.New(), IdempotencyKey: uuid.NewString(), PayloadHash: "h", SalesOrderCode: pedido, CreatedBy: actor,
		Lines: []reschedulerepo.BatchLine{{SalesOrderItemCode: linha, ItemCode: valueobject.ItemCode(item), OldDate: entrega, NewDate: nova}}}
	if _, err := repo.CreateBatch(b.Ctx, cmd); !errors.Is(err, reschedulerepo.ErrOrderNotFound) {
		t.Fatalf("empresa B reprogramou o pedido da A: %v", err)
	}
	lote, err := repo.CreateBatch(a.Ctx, cmd)
	if err != nil || len(lote.Codes) != 1 {
		t.Fatalf("reprogramação em lote na empresa do pedido: %+v %v", lote, err)
	}
	if again, err := repo.CreateBatch(a.Ctx, cmd); err != nil || !again.Replayed {
		t.Fatalf("repetição idempotente: %+v %v", again, err)
	}

	if _, err := repo.Create(b.Ctx, &entity.DeliveryReschedule{SalesOrderCode: pedido, ItemCode: valueobject.ItemCode(item), OldDate: nova, NewDate: nova.AddDate(0, 0, 1), CreatedBy: actor}); err == nil {
		t.Fatal("empresa B gravou reprogramação no pedido da A")
	}
	code := lote.Codes[0]
	if _, err := repo.GetByCode(b.Ctx, code); err == nil {
		t.Fatal("empresa B leu a reprogramação da A")
	}
	if l, err := repo.ListByOrder(b.Ctx, pedido); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d reprogramação(ões) da A (%v)", len(l), err)
	}
	if l, err := repo.ListByItem(b.Ctx, valueobject.ItemCode(item)); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d reprogramação(ões) do item na A (%v)", len(l), err)
	}
	if err := repo.Delete(b.Ctx, code); err == nil {
		t.Fatal("empresa B apagou a reprogramação da A")
	}
	if l, err := repo.ListByItem(a.Ctx, valueobject.ItemCode(item)); err != nil || len(l) != 1 {
		t.Fatalf("empresa A deveria ver a própria reprogramação: %d (%v)", len(l), err)
	}
	if got, err := repo.GetByCode(a.Ctx, code); err != nil || got.Code != code {
		t.Fatalf("empresa A não leu a própria reprogramação: %v", err)
	}
}
