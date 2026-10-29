//go:build integration

package purchase_order

import (
	"context"
	"strings"
	"testing"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Cancelar o pedido de compra cancela os títulos a pagar dele (adiantamento,
// título lançado com o pedido) e os tira da lista; título pago ou que veio da
// nota de entrada impede o cancelamento.
func TestCancelamentoDoPedidoDeCompraCancelaOsTitulos(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	empresaCode := int64(1_940_000_000 + u%50_000_000)
	var emp int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Cancelamento do pedido de compra') RETURNING id`, empresaCode).Scan(&emp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.ApagarEmpresasPorCodigo(t, pool, int64(empresaCode)) })
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: emp, EnterpriseCode: empresaCode})
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM contas_pagar WHERE enterprise_id=$1`, emp)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`, emp)
		_, _ = pool.Exec(bg, `DELETE FROM purchase_orders WHERE enterprise_code=$1`, empresaCode)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, emp)
	})
	pedido := func(n int) int64 {
		t.Helper()
		var code int64
		if err := pool.QueryRow(bg, `INSERT INTO purchase_orders(order_number,enterprise_code,created_by) VALUES($1,$2,$3) RETURNING code`, n, empresaCode, ator).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}
	titulo := func(po int64, doc, status, pago string, nota *int64) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO contas_pagar(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_pago,status,purchase_order_id,fiscal_entry_id,criado_por,enterprise_id)
			VALUES($1,$2::date,$2::date + 30,100,$3::numeric,$4,$5,$6,$7,$8) RETURNING id`, doc, time.Now(), pago, status, po, nota, ator, emp).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	situacao := func(id int64) (string, bool) {
		t.Helper()
		var st string
		var ativo bool
		if err := pool.QueryRow(bg, `SELECT UPPER(status), is_active FROM contas_pagar WHERE id=$1`, id).Scan(&st, &ativo); err != nil {
			t.Fatal(err)
		}
		return st, ativo
	}
	statusPedido := func(code int64) string {
		var st string
		if err := pool.QueryRow(bg, `SELECT status FROM purchase_orders WHERE code=$1`, code).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	repo := &PurchaseOrderRepositorySQLC{db: pool}

	t.Run("títulos em aberto caem com o pedido", func(t *testing.T) {
		po := pedido(1)
		a, b := titulo(po, "ADT-1", "ABERTO", "0", nil), titulo(po, "ADT-2", "ABERTO", "0", nil)
		if err := repo.Cancel(ctx, po); err != nil {
			t.Fatal(err)
		}
		for _, id := range []int64{a, b} {
			if st, ativo := situacao(id); st != "CANCELADO" || ativo {
				t.Fatalf("título %d: %s ativo=%v, quer CANCELADO inativo", id, st, ativo)
			}
		}
	})
	t.Run("título pago impede e nada muda", func(t *testing.T) {
		po := pedido(2)
		aberto, pago := titulo(po, "ADT-3", "ABERTO", "0", nil), titulo(po, "ADT-4", "PARCIAL", "40", nil)
		err := repo.Cancel(ctx, po)
		if err == nil || !strings.Contains(err.Error(), "já teve pagamento") {
			t.Fatalf("erro = %v, quer recusa por pagamento", err)
		}
		if st := statusPedido(po); st == "CANCELLED" {
			t.Fatal("o pedido ficou cancelado mesmo com a recusa")
		}
		for _, id := range []int64{aberto, pago} {
			if _, ativo := situacao(id); !ativo {
				t.Fatalf("título %d foi inativado apesar da recusa", id)
			}
		}
	})
	t.Run("título da nota de entrada é cancelado pela nota", func(t *testing.T) {
		po := pedido(3)
		var nota int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(numero_nf,serie,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,valor_produtos,valor_total,status,created_by,enterprise_id)
			VALUES(77,'1',$1::date,$1::date,'27785001000145','FORNECEDOR',100,100,'APPROVED',$2,$3) RETURNING id`, time.Now(), ator, emp).Scan(&nota); err != nil {
			t.Fatal(err)
		}
		titulo(po, "77/1", "ABERTO", "0", &nota)
		if err := repo.Cancel(ctx, po); err == nil || !strings.Contains(err.Error(), "veio da nota de entrada") {
			t.Fatalf("erro = %v, quer recusa pelo título da nota", err)
		}
	})
}
