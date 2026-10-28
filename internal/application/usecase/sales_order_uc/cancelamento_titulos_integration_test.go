//go:build integration

package sales_order_uc_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Cancelar o pedido de venda cancela os títulos a receber dele e os tira da
// lista; título recebido (ou que veio da NF de saída) impede o cancelamento,
// e o pedido continua como estava.
func TestCancelamentoDoPedidoDeVendaCancelaOsTitulos(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	empresaCode := int64(1_990_000_000 + u%9_000_000)
	var emp int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Cancelamento do pedido de venda') RETURNING id`, empresaCode).Scan(&emp); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: emp, EnterpriseCode: empresaCode})
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM contas_receber WHERE enterprise_id=$1`, emp)
		_, _ = pool.Exec(bg, `DELETE FROM sales_order_events WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`, empresaCode)
		_, _ = pool.Exec(bg, `DELETE FROM sales_orders WHERE enterprise_code=$1`, empresaCode)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, emp)
	})
	pedido := func(n int) int64 {
		t.Helper()
		var code int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,created_by) VALUES($1,$2,'C',$3) RETURNING code`, n, empresaCode, ator).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}
	titulo := func(so int64, doc, status, recebido string) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,sales_order_id,criado_por,enterprise_id)
			VALUES($1,$2::date,$2::date + 30,100,$3::numeric,$4,$5,$6,$7) RETURNING id`, doc, time.Now(), recebido, status, so, ator, emp).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	ativo := func(id int64) (string, bool) {
		t.Helper()
		var st string
		var a bool
		if err := pool.QueryRow(bg, `SELECT UPPER(status), is_active FROM contas_receber WHERE id=$1`, id).Scan(&st, &a); err != nil {
			t.Fatal(err)
		}
		return st, a
	}
	statusPedido := func(code int64) string {
		var st string
		if err := pool.QueryRow(bg, `SELECT status FROM sales_orders WHERE code=$1`, code).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	q, _ := testutil.Queries(t)
	uc := &sales_order_uc.CancelSalesOrderUseCase{
		Repo:    salesorderpg.NewSalesOrderRepositorySQLC(q),
		Auth:    &auth.AuthService{},
		Titulos: financialpg.NewFinancialRepositoryPG(pool).(sales_order_uc.RecebiveisDoPedido),
	}
	cancelar := func(so int64) error {
		return uc.Execute(ctx, request.CancelSalesOrderDTO{Code: so, Reason: "cliente desistiu"})
	}

	t.Run("títulos em aberto caem com o pedido", func(t *testing.T) {
		so := pedido(1)
		a, b := titulo(so, "SINAL-1", "ABERTO", "0"), titulo(so, "SINAL-2", "ABERTO", "0")
		if err := cancelar(so); err != nil {
			t.Fatal(err)
		}
		if st := statusPedido(so); st != "CANCELLED" {
			t.Fatalf("pedido = %s, quer CANCELLED", st)
		}
		for _, id := range []int64{a, b} {
			if st, at := ativo(id); st != "CANCELADO" || at {
				t.Fatalf("título %d: %s ativo=%v, quer CANCELADO inativo", id, st, at)
			}
		}
	})
	t.Run("título recebido impede e nada muda", func(t *testing.T) {
		so := pedido(2)
		aberto, recebido := titulo(so, "SINAL-3", "ABERTO", "0"), titulo(so, "SINAL-4", "PARCIAL", "30")
		err := cancelar(so)
		if err == nil || !strings.Contains(err.Error(), "já teve recebimento") {
			t.Fatalf("erro = %v, quer recusa por recebimento", err)
		}
		if st := statusPedido(so); st == "CANCELLED" {
			t.Fatal("o pedido ficou cancelado mesmo com a recusa")
		}
		for _, id := range []int64{aberto, recebido} {
			if _, at := ativo(id); !at {
				t.Fatalf("título %d foi inativado apesar da recusa", id)
			}
		}
	})
}
