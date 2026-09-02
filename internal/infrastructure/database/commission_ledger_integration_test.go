//go:build integration

package database_test

import (
	"context"
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
	if err := pool.QueryRow(ctx, `INSERT INTO sales_orders(order_number,enterprise_code,representative_code,commission_pct,total_net,created_by) VALUES(1,$1,77,5,1000,$2) RETURNING code`, enterpriseID, actor).Scan(&orderCode); err != nil {
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
