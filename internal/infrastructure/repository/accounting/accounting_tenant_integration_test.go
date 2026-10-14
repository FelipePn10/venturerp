//go:build integration

package accounting_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/accounting"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestAccountingIsIsolatedByTenant: plano de contas, contas e demonstrativos
// eram compartilhados por todas as empresas do banco — uma empresa via,
// alterava e lançava no plano da outra.
func TestAccountingIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ctx := context.Background()
		ids := []int64{a.ID, b.ID}
		for _, q := range []string{
			`DELETE FROM accounting_journal_entries WHERE empresa_id = ANY($1)`,
			`DELETE FROM accounting_account_refs WHERE empresa_id = ANY($1)`,
			`UPDATE accounting_accounts SET parent_id = NULL WHERE plan_id IN (SELECT id FROM accounting_plans WHERE enterprise_id = ANY($1))`,
			`DELETE FROM accounting_accounts WHERE plan_id IN (SELECT id FROM accounting_plans WHERE enterprise_id = ANY($1))`,
			`DELETE FROM accounting_plans WHERE enterprise_id = ANY($1)`,
			`DELETE FROM accounting_demonstrative_items WHERE demonstrative_id IN (SELECT id FROM accounting_demonstratives WHERE enterprise_id = ANY($1))`,
			`DELETE FROM accounting_demonstratives WHERE enterprise_id = ANY($1)`,
		} {
			if _, err := pool.Exec(ctx, q, ids); err != nil {
				t.Errorf("limpeza: %v", err)
			}
		}
	})
	repo := accounting.New(sqlc.New(pool), pool)
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	plano, err := repo.CreateAccountingPlan(a.Ctx, &entity.AccountingPlan{PlanNumber: 1, Description: "PLANO A", ValidFrom: inicio, Status: entity.PlanStatusAtivo})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetAccountingPlan(b.Ctx, plano.ID); err == nil {
		t.Fatal("empresa B leu o plano de contas da A")
	}
	if l, err := repo.ListAccountingPlans(b.Ctx); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d plano(s) da A (%v)", len(l), err)
	}
	plano.Description = "ALTERADO POR B"
	if _, err := repo.UpdateAccountingPlan(b.Ctx, plano); err == nil {
		t.Fatal("empresa B alterou o plano da A")
	}
	nova := func(num string, pai *int64) *entity.AccountingAccount {
		return &entity.AccountingAccount{PlanID: plano.ID, ParentID: pai, AccountNumber: num, Description: "CONTA " + num, NatureCode: "01", ValidFrom: inicio, IsAnalytic: pai != nil}
	}
	if _, err := repo.CreateAccountingAccount(b.Ctx, nova("1", nil)); err == nil {
		t.Fatal("empresa B criou conta no plano da A")
	}
	raiz, err := repo.CreateAccountingAccount(a.Ctx, nova("1", nil))
	if err != nil {
		t.Fatal(err)
	}
	deb, err := repo.CreateAccountingAccount(a.Ctx, nova("1.1", &raiz.ID))
	if err != nil {
		t.Fatal(err)
	}
	cred, err := repo.CreateAccountingAccount(a.Ctx, nova("1.2", &raiz.ID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetAccountingAccount(b.Ctx, deb.ID); err == nil {
		t.Fatal("empresa B leu conta da A")
	}
	if l, err := repo.ListAccountingAccountsByPlan(b.Ctx, plano.ID); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d conta(s) do plano da A (%v)", len(l), err)
	}
	deb.Description = "ALTERADA POR B"
	if _, err := repo.UpdateAccountingAccount(b.Ctx, deb); err == nil {
		t.Fatal("empresa B alterou conta da A")
	}

	// Plano próprio da B: não pode usar conta-pai nem contas da A.
	planoB, err := repo.CreateAccountingPlan(b.Ctx, &entity.AccountingPlan{PlanNumber: 1, Description: "PLANO B", ValidFrom: inicio, Status: entity.PlanStatusAtivo})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAccountingAccount(b.Ctx, &entity.AccountingAccount{PlanID: planoB.ID, ParentID: &raiz.ID, AccountNumber: "9", Description: "X", NatureCode: "01", ValidFrom: inicio}); err == nil {
		t.Fatal("empresa B pendurou conta na conta-pai da A")
	}
	lanc := &entity.AccountingJournalEntry{PlanID: plano.ID, EntryDate: inicio, EntryNumber: "L1", DebitAccountID: deb.ID, CreditAccountID: cred.ID, Value: 10, EntryType: "MANUAL"}
	if _, err := repo.CreateJournalEntry(b.Ctx, lanc); err == nil {
		t.Fatal("empresa B lançou no plano da A")
	}
	if _, err := repo.CreateJournalEntry(b.Ctx, &entity.AccountingJournalEntry{PlanID: planoB.ID, EntryDate: inicio, EntryNumber: "L1", DebitAccountID: deb.ID, CreditAccountID: cred.ID, Value: 10, EntryType: "MANUAL"}); err == nil {
		t.Fatal("empresa B lançou no próprio plano usando contas da A")
	}
	if _, err := repo.CreateJournalEntry(a.Ctx, lanc); err != nil {
		t.Fatalf("empresa A não lançou no próprio plano: %v", err)
	}
	if l, err := repo.ListJournalEntries(a.Ctx, plano.ID, 0, inicio, inicio); err != nil || len(l) != 1 {
		t.Fatalf("empresa A deveria ver o próprio lançamento: %d (%v)", len(l), err)
	}
	ref, err := repo.CreateReferenceAccount(a.Ctx, &entity.AccountingReferenceAccount{InstitutionCode: 1, AccountNumber: "TESTE-" + plano.Description, Description: "REFERENCIAL", AccountType: "A"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM accounting_account_refs WHERE ref_account_id=$1`, ref.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM accounting_reference_accounts WHERE id=$1`, ref.ID)
	})
	if _, err := repo.CreateAccountRef(b.Ctx, &entity.AccountingAccountRef{AccountID: deb.ID, RefAccountID: ref.ID}); err == nil {
		t.Fatal("empresa B vinculou conta referencial à conta da A")
	}
	if _, err := repo.CreateAccountRef(a.Ctx, &entity.AccountingAccountRef{AccountID: deb.ID, RefAccountID: ref.ID}); err != nil {
		t.Fatalf("empresa A não vinculou a própria conta: %v", err)
	}

	dem, err := repo.CreateDemonstrative(a.Ctx, &entity.AccountingDemonstrative{Code: "DRE", Description: "DRE A"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetDemonstrative(b.Ctx, dem.ID); err == nil {
		t.Fatal("empresa B leu o demonstrativo da A")
	}
	if l, err := repo.ListDemonstratives(b.Ctx); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d demonstrativo(s) da A (%v)", len(l), err)
	}
	if _, err := repo.CreateDemonstrativeItem(b.Ctx, &entity.AccountingDemonstrativeItem{DemonstrativeID: dem.ID, ItemCode: 1, Description: "X"}); err == nil {
		t.Fatal("empresa B incluiu item no demonstrativo da A")
	}
	if _, err := repo.CreateDemonstrative(b.Ctx, &entity.AccountingDemonstrative{Code: "DRE", Description: "DRE B"}); err != nil {
		t.Fatalf("o mesmo código de demonstrativo tem de poder existir na empresa B: %v", err)
	}
}
