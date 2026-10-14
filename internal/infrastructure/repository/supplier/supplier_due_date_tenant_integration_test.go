//go:build integration

package supplier_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	supplierrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O vencimento do fornecedor só aceita fornecedor e condição de pagamento da
// própria empresa — antes a condição vinha do corpo da requisição sem conferência.
func TestVencimentoDoFornecedorIsoladoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	code := int64(1_720_000_000 + testutil.UniqueCode()%10_000_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		_, _ = pool.Exec(base, `DELETE FROM supplier_due_dates WHERE supplier_id IN (SELECT id FROM suppliers WHERE code=$1)`, code)
		_, _ = pool.Exec(base, `DELETE FROM suppliers WHERE code=$1`, code)
		_, _ = pool.Exec(base, `DELETE FROM payment_conditions WHERE enterprise_id = ANY($1)`, ids)
	})
	var fornecedor, condA, condB int64
	if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_number,enterprise_id,created_by) VALUES($1,'FORNECEDOR DA A','00000000000191',$2,$3) RETURNING id`,
		code, a.ID, testutil.Actor(t, pool)).Scan(&fornecedor); err != nil {
		t.Fatal(err)
	}
	cond := func(e testutil.Empresa) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO payment_conditions(code,description,enterprise_id) VALUES($1,'30 DIAS',$2) RETURNING id`, testutil.UniqueCode()%1_000_000, e.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	condA, condB = cond(a), cond(b)
	repo := supplierrepo.New(sqlc.New(pool), pool)
	venc := func(cond int64) *entity.SupplierDueDate {
		return &entity.SupplierDueDate{SupplierID: fornecedor, Description: "Padrão", Ranking: 1, BaseDate: entity.BaseDateEmissao,
			PaymentConditionID: &cond, PaymentType: entity.DuePaymentNaoInformado, Rounding: entity.RoundingFixo}
	}
	if _, err := repo.AddDueDate(b.Ctx, venc(condB)); err == nil {
		t.Fatal("empresa B incluiu vencimento no fornecedor da A")
	}
	if _, err := repo.AddDueDate(a.Ctx, venc(condB)); err == nil {
		t.Fatal("empresa A apontou a condição de pagamento da B")
	}
	if _, err := repo.AddDueDate(a.Ctx, venc(condA)); err != nil {
		t.Fatalf("empresa A não incluiu o próprio vencimento: %v", err)
	}
}
