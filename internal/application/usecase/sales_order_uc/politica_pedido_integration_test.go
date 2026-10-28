//go:build integration

// Alçada de desconto do pedido (Fase 5.3): acima da alçada sem quem aprove a
// linha é recusada; com a aprovação PEDIDO_POLITICA_COMERCIAL ativa o pedido
// bloqueia e abre UMA solicitação; aprovar desbloqueia; editar de novo com o
// mesmo motivo não bloqueia outra vez, subir o desconto bloqueia. A alçada é
// da empresa: a outra não herda.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run AlcadaDeDesconto ./internal/application/usecase/sales_order_uc/
package sales_order_uc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/aprovacao_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/parametros_comerciais_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/domain/parametroscomerciais"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	aprovacaopg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aprovacao"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	parametrospg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/parametroscomerciais"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestAlcadaDeDescontoDoPedido(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		tx, err := pool.Begin(bg)
		if err != nil {
			return
		}
		defer tx.Rollback(bg)
		_, _ = tx.Exec(bg, `ALTER TABLE aprovacao_eventos DISABLE TRIGGER trg_aprovacao_eventos_imutaveis`)
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM aprovacao_eventos WHERE enterprise_id=$1`, `DELETE FROM aprovacao_solicitacoes WHERE enterprise_id=$1`,
				`DELETE FROM aprovacao_politicas WHERE enterprise_id=$1`, `DELETE FROM parametros_comerciais WHERE enterprise_id=$1`,
				`DELETE FROM notification_outbox WHERE enterprise_id=$1`,
				`DELETE FROM sales_order_events WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1))`,
				`DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1))`,
				`DELETE FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM customers WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`,
			} {
				if _, err := tx.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
		_, _ = tx.Exec(bg, `ALTER TABLE aprovacao_eventos ENABLE TRIGGER trg_aprovacao_eventos_imutaveis`)
		_ = tx.Commit(bg)
	})
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'ADMIN') ON CONFLICT DO NOTHING`, actor, a.ID)
	cliente, item := 866_000+u%90_000, 88_000_000+u%1_000_000
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE ALÇADA','11444777000161',$2,$3)`, cliente, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,'AL'||($1::bigint)::text,5,$2,$3)`, item, actor, a.ID)
	var pedido, linha int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by) VALUES(1,$1,'R',$2,$3) RETURNING code`, a.Code, cliente, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,discount_pct,total_gross,total_net,warehouse_code)
		VALUES($1,1,$2,10,10,15,100,85,5) RETURNING code`, pedido, item).Scan(&linha); err != nil {
		t.Fatal(err)
	}

	q, _ := testutil.Queries(t)
	repo := salesorderpg.NewSalesOrderRepositorySQLC(q)
	params := parametrospg.New(pool)
	parametrosUC := &parametros_comerciais_uc.UseCase{Repo: params, Auth: &auth.AuthService{}}
	aprov := &aprovacao_uc.UseCase{Repo: aprovacaopg.New(pool), Auth: &auth.AuthService{}}
	transacao := func(ctx context.Context, fn func(sales_order_uc.ConfirmacaoRepos) error) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := fn(sales_order_uc.ConfirmacaoRepos{Pedidos: salesorderpg.NewSalesOrderRepositorySQLC(sqlc.New(tx))}); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	conclusao := &sales_order_uc.ConclusaoDoDesbloqueio{Transacao: transacao}
	aprov.Registrar(aprovacao.PedidoPoliticaComercial, &sales_order_uc.LiberacaoDeCredito{Conclusao: conclusao, Tipo: aprovacao.PedidoPoliticaComercial, Aprovacoes: aprov})
	aprov.Registrar(aprovacao.LiberacaoCredito, &sales_order_uc.LiberacaoDeCredito{Conclusao: conclusao, Tipo: aprovacao.LiberacaoCredito, Aprovacoes: aprov})
	desbloquear := &sales_order_uc.UnblockSalesOrderUseCase{Repo: repo, Auth: &auth.AuthService{}, Aprovacoes: aprov, Conclusao: conclusao}
	politica := &sales_order_uc.PoliticaDoPedido{Repo: repo, Customers: customerpg.New(q, pool), Aprovacoes: aprov, Parametros: params}

	bloqueado := func() bool {
		t.Helper()
		var bl bool
		if err := pool.QueryRow(bg, `SELECT is_blocked FROM sales_orders WHERE code=$1`, pedido).Scan(&bl); err != nil {
			t.Fatal(err)
		}
		return bl
	}
	solicitacoes := func() (n int) {
		_ = pool.QueryRow(bg, `SELECT count(*) FROM aprovacao_solicitacoes WHERE enterprise_id=$1 AND tipo='PEDIDO_POLITICA_COMERCIAL'`, a.ID).Scan(&n)
		return
	}

	// Sem alçada: tudo passa.
	if err := politica.ConferirDesconto(a.Ctx, 15); err != nil {
		t.Fatal(err)
	}
	if err := politica.Aplicar(a.Ctx, pedido); err != nil || bloqueado() {
		t.Fatalf("sem alçada não bloqueia: %v", err)
	}

	cem := decimal.NewFromInt(101)
	if _, err := parametrosUC.Salvar(a.Ctx, parametroscomerciais.Parametros{DescontoMaxPct: &cem}); err == nil {
		t.Fatal("alçada acima de 100% é recusada")
	}
	dez := decimal.NewFromInt(10)
	if p, err := parametrosUC.Salvar(a.Ctx, parametroscomerciais.Parametros{DescontoMaxPct: &dez}); err != nil || p.DescontoMaxPct == nil || !p.DescontoMaxPct.Equal(dez) {
		t.Fatalf("salvar a alçada: %+v %v", p, err)
	}

	// Acima da alçada sem política de aprovação: recusa a linha.
	if err := politica.ConferirDesconto(a.Ctx, 15); err == nil {
		t.Fatal("15% acima de 10% sem quem aprove deveria ser recusado")
	}
	if err := politica.ConferirDesconto(a.Ctx, 10); err != nil {
		t.Fatalf("10%% está na alçada: %v", err)
	}

	// Com a aprovação ativa: passa, bloqueia e abre uma solicitação só.
	ator := actor.String()
	if err := aprov.SalvarPolitica(a.Ctx, string(aprovacao.PedidoPoliticaComercial), aprovacao_uc.PoliticaDTO{
		Ativa: true, Niveis: []aprovacao_uc.NivelDTO{{UserID: &ator}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := politica.ConferirDesconto(a.Ctx, 15); err != nil {
		t.Fatal(err)
	}
	if err := politica.Aplicar(a.Ctx, pedido); err != nil {
		t.Fatal(err)
	}
	if err := politica.Aplicar(a.Ctx, pedido); err != nil {
		t.Fatal(err)
	}
	if !bloqueado() || solicitacoes() != 1 {
		t.Fatalf("bloqueado=%v solicitações=%d", bloqueado(), solicitacoes())
	}
	pend, err := aprov.PendenteDoDocumento(a.Ctx, aprovacao.PedidoPoliticaComercial, pedido)
	if err != nil || pend == nil {
		t.Fatalf("pendente: %v", err)
	}
	if _, err := aprov.Aprovar(a.Ctx, pend.ID, aprovacao_uc.DecisaoDTO{Justificativa: "cliente estratégico"}); err != nil {
		t.Fatal(err)
	}
	if bloqueado() {
		t.Fatal("aprovar desbloqueia o pedido")
	}

	// Mesma situação: a aprovação concedida vale; subir o desconto pede outra.
	if err := politica.Aplicar(a.Ctx, pedido); err != nil || bloqueado() || solicitacoes() != 1 {
		t.Fatalf("aprovado não volta a bloquear: %v bloqueado=%v n=%d", err, bloqueado(), solicitacoes())
	}
	testutil.Exec(t, pool, `UPDATE sales_order_items SET discount_pct=20 WHERE code=$1`, linha)
	if err := politica.Aplicar(a.Ctx, pedido); err != nil || !bloqueado() || solicitacoes() != 2 {
		t.Fatalf("desconto maior pede nova aprovação: %v bloqueado=%v n=%d", err, bloqueado(), solicitacoes())
	}

	// O desbloqueio manual não contorna a alçada.
	if err := desbloquear.Execute(a.Ctx, pedido); err == nil || !strings.Contains(err.Error(), "política comercial") {
		t.Fatalf("desbloqueio manual com a alçada pendente deveria ser recusado: %v", err)
	}

	// Crédito E alçada pendentes no mesmo pedido: aprovar um não libera o
	// pedido enquanto o outro espera.
	if err := aprov.SalvarPolitica(a.Ctx, string(aprovacao.LiberacaoCredito), aprovacao_uc.PoliticaDTO{
		Ativa: true, Niveis: []aprovacao_uc.NivelDTO{{UserID: &ator}},
	}); err != nil {
		t.Fatal(err)
	}
	credito, err := aprov.Solicitar(a.Ctx, aprovacao.LiberacaoCredito, pedido, decimal.NewFromInt(80), "pedido bloqueado pelo limite de crédito")
	if err != nil || credito == nil {
		t.Fatalf("crédito: %v", err)
	}
	alcada, _ := aprov.PendenteDoDocumento(a.Ctx, aprovacao.PedidoPoliticaComercial, pedido)
	if _, err := aprov.Aprovar(a.Ctx, alcada.ID, aprovacao_uc.DecisaoDTO{}); err != nil {
		t.Fatal(err)
	}
	if !bloqueado() {
		t.Fatal("com o crédito pendente, aprovar a alçada não libera o pedido")
	}
	if err := desbloquear.Execute(a.Ctx, pedido); err == nil || !strings.Contains(err.Error(), "crédito") {
		t.Fatalf("desbloqueio manual com crédito pendente deveria ser recusado: %v", err)
	}
	if _, err := aprov.Aprovar(a.Ctx, credito.ID, aprovacao_uc.DecisaoDTO{}); err != nil {
		t.Fatal(err)
	}
	if bloqueado() {
		t.Fatal("aprovadas as duas, o pedido é liberado")
	}

	// A alçada é da empresa A.
	if p, err := parametrosUC.Obter(b.Ctx); err != nil || p.DescontoMaxPct != nil {
		t.Fatalf("a empresa B não herda a alçada: %+v %v", p, err)
	}
	if err := politica.ConferirDesconto(b.Ctx, 50); err != nil {
		t.Fatalf("sem alçada na B: %v", err)
	}
}
