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

// TestSupplierContactWritesAreIsolatedByTenant: a sonda cruzada incluiu
// telefone e e-mail no contato do fornecedor de OUTRA empresa pelo id.
func TestSupplierContactWritesAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	code := int64(1_710_000_000 + testutil.UniqueCode()%10_000_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM suppliers WHERE code=$1`, code)
	})
	var supplierID, contato int64
	if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_number,enterprise_id,created_by) VALUES($1,'FORNECEDOR DA A','00000000000191',$2,$3) RETURNING id`,
		code, a.ID, testutil.Actor(t, pool)).Scan(&supplierID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO supplier_contacts(supplier_id,name) VALUES($1,'Contato da A') RETURNING id`, supplierID).Scan(&contato); err != nil {
		t.Fatal(err)
	}
	repo := supplierrepo.New(sqlc.New(pool), pool)

	if _, err := repo.AddContactPhone(b.Ctx, &entity.SupplierContactPhone{ContactID: contato, Value: "4199999999", Ranking: 1}); err == nil {
		t.Fatal("empresa B incluiu telefone no contato do fornecedor da A")
	}
	if _, err := repo.AddContactEmail(b.Ctx, &entity.SupplierContactEmail{ContactID: contato, Value: "b@b.com", Ranking: 1}); err == nil {
		t.Fatal("empresa B incluiu e-mail no contato do fornecedor da A")
	}
	if _, err := repo.AddContactPhone(a.Ctx, &entity.SupplierContactPhone{ContactID: contato, Value: "4188888888", Ranking: 1}); err != nil {
		t.Fatalf("empresa A não incluiu telefone no próprio contato: %v", err)
	}
	if _, err := repo.AddContactEmail(a.Ctx, &entity.SupplierContactEmail{ContactID: contato, Value: "a@a.com", Ranking: 1}); err != nil {
		t.Fatalf("empresa A não incluiu e-mail no próprio contato: %v", err)
	}
}
