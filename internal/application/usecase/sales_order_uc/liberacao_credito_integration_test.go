//go:build integration

// Pedido bloqueado pelo crédito passa pela alçada: abre a aprovação, o
// desbloqueio manual é recusado, quem confirmou não aprova (segregação) e, ao
// aprovar, o pedido é liberado E termina a confirmação — demanda do MRP e
// reserva de estoque, que antes nunca eram geradas depois do desbloqueio.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run LiberacaoDeCredito ./internal/application/usecase/sales_order_uc/
package sales_order_uc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/aprovacao_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	aprovacaopg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aprovacao"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestLiberacaoDeCreditoPelaAprovacao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	empresaCode := int64(1_960_000_000 + u%30_000_000)
	var empresaID, pedido, perfil int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Liberação de crédito') RETURNING id`, empresaCode).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	aprovador := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password,role,auth_version,is_active) VALUES($1,'Aprovador de crédito',$2,'x','USER',1,true)`, aprovador, aprovador.String()+"@test.local")
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'ADMIN'),($3,$2,'USER')`, actor, empresaID, aprovador)
	if err := pool.QueryRow(bg, `INSERT INTO acesso_perfis(enterprise_id,nome,created_by,updated_by) VALUES($1,'Crédito',$2,$2) RETURNING id`, empresaID, actor).Scan(&perfil); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO acesso_usuario_perfis(enterprise_id,user_id,perfil_id,created_by) VALUES($1,$2,$3,$4)`, empresaID, aprovador, perfil, actor)

	cliente, item := u, u
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,credit_limit,created_by,enterprise_id) VALUES($1,'CLIENTE COM LIMITE','12345678000195',10,$2,$3)`, cliente, actor, empresaID)
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,5,$2,$3)`, item, actor, empresaID)
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,total_net,created_by) VALUES(1,$1,'R',$2,40,$3) RETURNING code`, empresaCode, cliente, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net,warehouse_code) VALUES($1,1,$2,4,10,40,5)`, pedido, item)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,enterprise_id) VALUES($1,5,10,0,$2)`, item, empresaID)
	t.Cleanup(func() {
		tx, err := pool.Begin(bg)
		if err != nil {
			return
		}
		defer tx.Rollback(bg)
		_, _ = tx.Exec(bg, `ALTER TABLE aprovacao_eventos DISABLE TRIGGER trg_aprovacao_eventos_imutaveis`)
		_, _ = tx.Exec(bg, `DELETE FROM aprovacao_eventos WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `ALTER TABLE aprovacao_eventos ENABLE TRIGGER trg_aprovacao_eventos_imutaveis`)
		_, _ = tx.Exec(bg, `DELETE FROM aprovacao_solicitacoes WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM aprovacao_politicas WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM stock_reservations WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM stock_balances WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM independent_demands WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM sales_order_events WHERE sales_order_code=$1`, pedido)
		_, _ = tx.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, pedido)
		_, _ = tx.Exec(bg, `DELETE FROM sales_orders WHERE code=$1`, pedido)
		_, _ = tx.Exec(bg, `DELETE FROM customers WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM items WHERE code=$1`, item)
		_, _ = tx.Exec(bg, `DELETE FROM acesso_perfis WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM user_enterprises WHERE enterprise_id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresaID)
		_, _ = tx.Exec(bg, `DELETE FROM users WHERE id=$1`, aprovador)
		_ = tx.Commit(bg)
	})

	sessao := func(user uuid.UUID, papel string) context.Context {
		return context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: user.String(), Role: papel, EnterpriseID: empresaID, EnterpriseCode: empresaCode})
	}
	vendedor, credito := sessao(actor, "ADMIN"), sessao(aprovador, "USER")

	q, _ := testutil.Queries(t)
	repo := salesorderpg.NewSalesOrderRepositorySQLC(q)
	transacao := func(ctx context.Context, fn func(sales_order_uc.ConfirmacaoRepos) error) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		txq := sqlc.New(tx)
		if err := fn(sales_order_uc.ConfirmacaoRepos{
			Pedidos: salesorderpg.NewSalesOrderRepositorySQLC(txq),
			Estoque: stockpg.NewStockRepositoryTx(tx),
		}); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	aprov := &aprovacao_uc.UseCase{Repo: aprovacaopg.New(pool), Auth: &auth.AuthService{}}
	conclusao := &sales_order_uc.ConclusaoDoDesbloqueio{Transacao: transacao, Reserver: &sales_order_uc.OrderStockReserver{}}
	aprov.Registrar(aprovacao.LiberacaoCredito, &sales_order_uc.LiberacaoDeCredito{Conclusao: conclusao})
	if err := aprov.SalvarPolitica(vendedor, string(aprovacao.LiberacaoCredito), aprovacao_uc.PoliticaDTO{
		Ativa: true, Segregacao: true, Niveis: []aprovacao_uc.NivelDTO{{PerfilID: &perfil}},
	}); err != nil {
		t.Fatal(err)
	}
	confirmar := &sales_order_uc.ChangeStatusSalesOrderUseCase{
		Repo: repo, Auth: &auth.AuthService{},
		Reserver: &sales_order_uc.OrderStockReserver{}, Transacao: transacao, Aprovacoes: aprov,
		CreditChecker: &sales_order_uc.CreditChecker{SalesRepo: repo, CustomerRepo: customerpg.New(q, pool), FinancialRepo: financialpg.NewFinancialRepositoryPG(pool)},
	}
	desbloquear := &sales_order_uc.UnblockSalesOrderUseCase{Repo: repo, Auth: &auth.AuthService{}, Aprovacoes: aprov, Conclusao: conclusao}

	situacao := func() (bloqueado bool, demandas, reservas int) {
		t.Helper()
		if err := pool.QueryRow(bg, `SELECT is_blocked FROM sales_orders WHERE code=$1`, pedido).Scan(&bloqueado); err != nil {
			t.Fatal(err)
		}
		_ = pool.QueryRow(bg, `SELECT count(*) FROM independent_demands WHERE enterprise_id=$1`, empresaID).Scan(&demandas)
		_ = pool.QueryRow(bg, `SELECT count(*) FROM stock_reservations WHERE enterprise_id=$1 AND status='ACTIVE'`, empresaID).Scan(&reservas)
		return
	}

	// 1. Confirmar acima do limite: bloqueia e abre a aprovação.
	if err := confirmar.Execute(vendedor, request.ChangeStatusDTO{Code: pedido, Status: "P"}); err != nil {
		t.Fatal(err)
	}
	if b, d, r := situacao(); !b || d != 0 || r != 0 {
		t.Fatalf("bloqueado pelo crédito: bloqueado=%v demandas=%d reservas=%d", b, d, r)
	}
	pendente, err := aprov.PendenteDoDocumento(vendedor, aprovacao.LiberacaoCredito, pedido)
	if err != nil || pendente == nil || pendente.NiveisExigidos != 1 || !pendente.Valor.Equal(pendente.Valor.Round(2)) {
		t.Fatalf("a aprovação deveria estar aberta: %+v %v", pendente, err)
	}
	// Reconfirmar não duplica a solicitação.
	_ = confirmar.Execute(vendedor, request.ChangeStatusDTO{Code: pedido, Status: "P"})
	var abertas int
	_ = pool.QueryRow(bg, `SELECT count(*) FROM aprovacao_solicitacoes WHERE enterprise_id=$1`, empresaID).Scan(&abertas)
	if abertas != 1 {
		t.Fatalf("deveria haver uma solicitação, há %d", abertas)
	}

	// 2. Desbloqueio manual recusado com a aprovação pendente.
	if err := desbloquear.Execute(vendedor, pedido); err == nil || !strings.Contains(err.Error(), "aguarda a aprovação de crédito") {
		t.Fatalf("desbloqueio manual deveria ser recusado: %v", err)
	}

	// 3. Quem confirmou não aprova (segregação); quem não tem o perfil também não.
	if _, err := aprov.Aprovar(vendedor, pendente.ID, aprovacao_uc.DecisaoDTO{}); err == nil || !strings.Contains(err.Error(), "segregação") {
		t.Fatalf("quem pediu não pode aprovar: %v", err)
	}
	lista, err := aprov.ListarSolicitacoes(credito, "PENDENTE")
	if err != nil || len(lista) != 1 || !lista[0].PossoAprovar {
		t.Fatalf("o aprovador de crédito deveria ver a pendente como sua: %+v %v", lista, err)
	}

	// 4. O aprovador aprova: pedido liberado e confirmação concluída.
	feita, err := aprov.Aprovar(credito, pendente.ID, aprovacao_uc.DecisaoDTO{Justificativa: "cliente pagou o título vencido"})
	if err != nil {
		t.Fatal(err)
	}
	if feita.Status != "APROVADA" || len(feita.Eventos) != 2 || feita.Eventos[1].Evento != "APROVADA" {
		t.Fatalf("aprovação concluída: %+v", feita)
	}
	if b, d, r := situacao(); b || d != 0 || r != 1 {
		t.Fatalf("liberado pela aprovação: bloqueado=%v demandas=%d reservas=%d, quer false/0/1 (o MRP lê o pedido direto)", b, d, r)
	}
	if _, err := aprov.Aprovar(credito, pendente.ID, aprovacao_uc.DecisaoDTO{}); err == nil {
		t.Fatal("aprovação concluída não se aprova de novo")
	}

	// 5. O histórico não se altera.
	if _, err := pool.Exec(bg, `UPDATE aprovacao_eventos SET justificativa='editado' WHERE solicitacao_id=$1`, pendente.ID); err == nil {
		t.Fatal("evento de aprovação não pode ser alterado")
	}
}
