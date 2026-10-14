//go:build integration

// Confirmação do pedido é tudo ou nada: status, demanda do MRP e reserva de
// estoque entram juntos. Antes a reserva (e a demanda) falhavam em silêncio e o
// pedido ficava confirmado sem elas.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run ConfirmacaoDoPedido ./internal/application/usecase/sales_order_uc/
package sales_order_uc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	demandpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type reservaQueFalha struct{ stockrepo.StockRepository }

func (reservaQueFalha) CreateReservation(context.Context, *stockentity.StockReservation) (*stockentity.StockReservation, error) {
	return nil, errors.New("depósito em inventário")
}

func TestConfirmacaoDoPedidoTudoOuNada(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	empresaCode := int64(1_930_000_000 + u%60_000_000)
	var empresaID, pedido int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Confirmação do pedido') RETURNING id`, empresaCode).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresaID, EnterpriseCode: empresaCode})
	item := u
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,5,$2,$3)`, item, actor, empresaID)
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,created_by) VALUES(1,$1,'R',$2) RETURNING code`, empresaCode, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net,warehouse_code)
		VALUES($1,1,$2,4,10,40,5)`, pedido, item)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,enterprise_id) VALUES($1,5,10,0,$2)`, item, empresaID)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM stock_reservations WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM stock_balances WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM independent_demands WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM sales_order_events WHERE sales_order_code=$1`, pedido)
		_, _ = pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, pedido)
		_, _ = pool.Exec(bg, `DELETE FROM sales_orders WHERE code=$1`, pedido)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE code=$1`, item)
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresaID)
	})

	q, _ := testutil.Queries(t)
	falhar := true
	uc := &sales_order_uc.ChangeStatusSalesOrderUseCase{
		Repo:       salesorderpg.NewSalesOrderRepositorySQLC(q),
		Auth:       &auth.AuthService{},
		DemandRepo: demandpg.NewIndependentDemandRepositorySQLC(q),
		Reserver:   &sales_order_uc.OrderStockReserver{},
		Transacao: func(ctx context.Context, fn func(sales_order_uc.ConfirmacaoRepos) error) error {
			tx, err := pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback(ctx) }()
			txq := sqlc.New(tx)
			var estoque stockrepo.StockRepository = stockpg.NewStockRepositoryTx(tx)
			if falhar {
				estoque = reservaQueFalha{estoque}
			}
			if err := fn(sales_order_uc.ConfirmacaoRepos{
				Pedidos:  salesorderpg.NewSalesOrderRepositorySQLC(txq),
				Demandas: demandpg.NewIndependentDemandRepositorySQLC(txq),
				Estoque:  estoque,
			}); err != nil {
				return err
			}
			return tx.Commit(ctx)
		},
	}
	situacao := func() (status string, demandas, reservas int) {
		t.Helper()
		if err := pool.QueryRow(bg, `SELECT status FROM sales_orders WHERE code=$1`, pedido).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `SELECT count(*) FROM independent_demands WHERE enterprise_id=$1`, empresaID).Scan(&demandas); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `SELECT count(*) FROM stock_reservations WHERE enterprise_id=$1 AND status='ACTIVE'`, empresaID).Scan(&reservas); err != nil {
			t.Fatal(err)
		}
		return
	}

	err := uc.Execute(ctx, request.ChangeStatusDTO{Code: pedido, Status: "P"})
	if err == nil || !strings.Contains(err.Error(), "depósito em inventário") {
		t.Fatalf("erro = %v, quer a falha da reserva", err)
	}
	if st, d, r := situacao(); st != "R" || d != 0 || r != 0 {
		t.Fatalf("depois da falha: status=%s demandas=%d reservas=%d, quer R/0/0 (nada pode ficar)", st, d, r)
	}

	falhar = false
	if err := uc.Execute(ctx, request.ChangeStatusDTO{Code: pedido, Status: "P"}); err != nil {
		t.Fatal(err)
	}
	if st, d, r := situacao(); st != "P" || d != 1 || r != 1 {
		t.Fatalf("confirmado: status=%s demandas=%d reservas=%d, quer P/1/1", st, d, r)
	}
}
