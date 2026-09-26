//go:build integration

package database_test

import (
	"context"
	"strconv"
	"testing"

	commissionentity "github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/entity"
	commissionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/commercial_commission"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
)

func TestCommissionLedgerAccruesOnceAndReversesOnFiscalCancellation(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := uuid.New()
	enterpriseCode := int64(1_800_000_000 + testutil.UniqueCode()%100_000_000)
	var enterpriseID, orderCode, fiscalID int64
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Commission Integration',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name) VALUES($1,'Commission') RETURNING id`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	// `sales_orders.enterprise_code` é o CÓDIGO da empresa, não o id. O teste
	// gravava o id aqui e passava porque o gatilho comparava as duas chaves
	// direto — o que só acerta quando id = código (a empresa 1 do dev).
	if err := pool.QueryRow(ctx, `INSERT INTO sales_orders(order_number,enterprise_code,representative_code,commission_pct,total_net,created_by) VALUES(1,$1,77,5,1000,$2) RETURNING code`, enterpriseCode, actor).Scan(&orderCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,valor_total,sales_order_code,status,created_by,enterprise_id) VALUES(1,CURRENT_DATE,'5102','Venda',1000,$1,'DRAFT',$2,$3) RETURNING id`, orderCode, actor, enterpriseID).Scan(&fiscalID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM notification_outbox WHERE originator_user_id=$1`, actor)
		_, _ = pool.Exec(ctx, `DELETE FROM commercial_commission_events WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM commercial_commission_settings WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM fiscal_exits WHERE id=$1`, fiscalID)
		_, _ = pool.Exec(ctx, `DELETE FROM sales_orders WHERE code=$1`, orderCode)
		_, _ = pool.Exec(ctx, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, actor)
	})
	if _, err := pool.Exec(ctx, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, fiscalID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, fiscalID); err != nil {
		t.Fatal(err)
	}
	var count int
	var ledgerCode int64
	var amount float64
	if err := pool.QueryRow(ctx, `SELECT count(*),sum(amount) FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID).Scan(&count, &amount); err != nil {
		t.Fatal(err)
	}
	if count != 1 || amount != 50 {
		t.Fatalf("competência duplicada/incorreta: count=%d amount=%.2f", count, amount)
	}
	if err := pool.QueryRow(ctx, `SELECT code FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID).Scan(&ledgerCode); err != nil {
		t.Fatal(err)
	}
	repo := commissionrepo.New(pool)
	reconciled, err := repo.Transition(ctx, enterpriseID, commissionentity.TransitionCommand{Code: ledgerCode, Action: "CONCILIADA", Reason: "documentos conferidos", IdempotencyKey: "commission-reconcile", ActorID: actor})
	if err != nil || reconciled.Status != "CONCILIADO" {
		t.Fatalf("conciliação falhou: status=%v err=%v", reconciled, err)
	}
	ref := "PAG-001"
	paid, err := repo.Transition(ctx, enterpriseID, commissionentity.TransitionCommand{Code: ledgerCode, Action: "PAGA", Reason: "pagamento aprovado", PaymentReference: &ref, IdempotencyKey: "commission-payment", ActorID: actor})
	if err != nil || paid.Status != "PAGO" {
		t.Fatalf("pagamento falhou: status=%v err=%v", paid, err)
	}
	repeated, err := repo.Transition(ctx, enterpriseID, commissionentity.TransitionCommand{Code: ledgerCode, Action: "PAGA", Reason: "pagamento aprovado", PaymentReference: &ref, IdempotencyKey: "commission-payment", ActorID: actor})
	if err != nil || repeated.Code != paid.Code {
		t.Fatalf("pagamento não foi idempotente: result=%v err=%v", repeated, err)
	}
	var eventCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM commercial_commission_events WHERE enterprise_id=$1 AND ledger_code=$2`, enterpriseID, ledgerCode).Scan(&eventCount); err != nil || eventCount != 2 {
		t.Fatalf("auditoria de comissão incompleta: count=%d err=%v", eventCount, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fiscal_exits SET status='CANCELLED' WHERE id=$1`, fiscalID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*),sum(amount) FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID).Scan(&count, &amount); err != nil {
		t.Fatal(err)
	}
	if count != 2 || amount != 0 {
		t.Fatalf("estorno incorreto: count=%d amount=%.2f", count, amount)
	}
}

// A razão de comissões tem de pagar TODOS os representantes do rateio, não só o
// da capa. Sem isso o parceiro aparecia no pedido e nunca recebia — a comissão
// dele continuaria sendo acertada por fora, que é o que o rateio veio resolver.
func TestCommissionLedgerAccruesForEveryRepresentativeInTheSplit(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := uuid.New()
	enterpriseCode := int64(1_800_000_000 + testutil.UniqueCode()%100_000_000)
	var enterpriseID, orderCode, fiscalID int64
	principal, parceiro := testutil.UniqueCode(), testutil.UniqueCode()

	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Rateio Integration',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name) VALUES($1,'Rateio') RETURNING id`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	for _, code := range []int64{principal, parceiro} {
		documento := strconv.FormatInt(code, 10)
		if _, err := pool.Exec(ctx, `INSERT INTO representatives(code,name,document_number,is_active,blocked) VALUES($1,'Rep '||$2,$2,TRUE,FALSE)`, code, documento); err != nil {
			t.Fatal(err)
		}
	}
	// A capa guarda o principal (é o espelho); o rateio guarda os dois.
	// O pedido e o rateio guardam o CÓDIGO da empresa; a nota guarda o ID. Usar
	// cada chave no seu lugar é o que prova a correção de convenção: antes o
	// gatilho comparava os dois direto e só acertava na empresa em que id = código.
	if err := pool.QueryRow(ctx, `INSERT INTO sales_orders(order_number,enterprise_code,representative_code,commission_pct,total_net,created_by) VALUES(1,$1,$2,3,1000,$3) RETURNING code`, enterpriseCode, principal, actor).Scan(&orderCode); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO sales_order_representatives(enterprise_code,sales_order_code,representative_code,role,commission_pct,commission_base) VALUES
		($1,$2,$3,'PRINCIPAL',3,'TOTAL_PRODUTOS'),($1,$2,$4,'PARCEIRO',1.5,'TOTAL_LIQUIDO')`, enterpriseCode, orderCode, principal, parceiro); err != nil {
		t.Fatal(err)
	}
	// Nota com produtos (800) diferente do total (1000): é o que prova que a
	// base gravada em cada linha do rateio foi respeitada.
	if err := pool.QueryRow(ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,valor_produtos,valor_total,sales_order_code,status,created_by,enterprise_id) VALUES(1,CURRENT_DATE,'5102','Venda',800,1000,$1,'DRAFT',$2,$3) RETURNING id`, orderCode, actor, enterpriseID).Scan(&fiscalID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM notification_outbox WHERE originator_user_id=$1`, actor)
		_, _ = pool.Exec(ctx, `DELETE FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM fiscal_exits WHERE id=$1`, fiscalID)
		_, _ = pool.Exec(ctx, `DELETE FROM sales_order_representatives WHERE sales_order_code=$1`, orderCode)
		_, _ = pool.Exec(ctx, `DELETE FROM sales_orders WHERE code=$1`, orderCode)
		_, _ = pool.Exec(ctx, `DELETE FROM representatives WHERE code=ANY($1)`, []int64{principal, parceiro})
		_, _ = pool.Exec(ctx, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, actor)
	})

	if _, err := pool.Exec(ctx, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, fiscalID); err != nil {
		t.Fatal(err)
	}
	// Autorizar duas vezes não pode duplicar nada (idempotência por linha).
	if _, err := pool.Exec(ctx, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, fiscalID); err != nil {
		t.Fatal(err)
	}

	// 3% sobre 800 (produtos) = 24; 1,5% sobre 1000 (líquido) = 15.
	var principalAmount, parceiroAmount float64
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("a razão deveria ter um lançamento por representante, tem %d", count)
	}
	if err := pool.QueryRow(ctx, `SELECT amount FROM commercial_commission_ledger WHERE enterprise_id=$1 AND representative_code=$2`, enterpriseID, principal).Scan(&principalAmount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT amount FROM commercial_commission_ledger WHERE enterprise_id=$1 AND representative_code=$2`, enterpriseID, parceiro).Scan(&parceiroAmount); err != nil {
		t.Fatal(err)
	}
	if principalAmount != 24 {
		t.Fatalf("principal: base de PRODUTOS não respeitada, amount=%.2f (esperado 24)", principalAmount)
	}
	if parceiroAmount != 15 {
		t.Fatalf("parceiro: base LÍQUIDA não respeitada, amount=%.2f (esperado 15)", parceiroAmount)
	}

	// Cancelar a nota estorna os DOIS lançamentos, não só o do principal.
	if _, err := pool.Exec(ctx, `UPDATE fiscal_exits SET status='CANCELLED' WHERE id=$1`, fiscalID); err != nil {
		t.Fatal(err)
	}
	var estornos int
	var saldo float64
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(SUM(amount),0) FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID).Scan(&estornos, &saldo); err != nil {
		t.Fatal(err)
	}
	if estornos != 4 || saldo != 0 {
		t.Fatalf("estorno incompleto: %d lançamento(s), saldo %.2f (esperado 4 e 0)", estornos, saldo)
	}
	var abertos int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM commercial_commission_ledger WHERE enterprise_id=$1 AND event_type='COMPETENCIA_FATURAMENTO' AND status<>'ESTORNADO'`, enterpriseID).Scan(&abertos); err != nil {
		t.Fatal(err)
	}
	if abertos != 0 {
		t.Fatalf("%d competência(s) continuaram abertas depois do cancelamento", abertos)
	}
}
