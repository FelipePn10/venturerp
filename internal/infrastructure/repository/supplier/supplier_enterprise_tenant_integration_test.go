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

// TestSupplierEnterpriseLinkIsIsolatedByTenant: a alteração e a exclusão do
// vínculo fornecedor×empresa iam pelo id, sem olhar a empresa; e a inclusão
// aceitava o fornecedor e a tabela de preço de compra de outra empresa.
func TestSupplierEnterpriseLinkIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		_, _ = pool.Exec(base, `DELETE FROM supplier_enterprises WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		_, _ = pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(base, `DELETE FROM purchase_price_tables WHERE enterprise_id = ANY($1)`, ids)
	})
	fornecedor := func(e testutil.Empresa, doc string) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_number,enterprise_id,created_by) VALUES($1,'FORNECEDOR',$2,$3,$4) RETURNING id`,
			testutil.UniqueCode(), doc, e.ID, actor).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	fornA, fornB := fornecedor(a, "00000000000191"), fornecedor(b, "00000000000272")
	var tabelaB int64
	if err := pool.QueryRow(base, `INSERT INTO purchase_price_tables(enterprise_id,code,description,created_by) VALUES($1,$2,'TABELA B',$3) RETURNING id`,
		b.ID, testutil.UniqueCode(), actor).Scan(&tabelaB); err != nil {
		t.Fatal(err)
	}
	repo := supplierrepo.New(sqlc.New(pool), pool)

	if _, err := repo.AddEnterprise(a.Ctx, &entity.SupplierEnterprise{SupplierID: fornB, EnterpriseCode: a.Code}); err == nil {
		t.Fatal("a A vinculou o fornecedor da B")
	}
	if _, err := repo.AddEnterprise(a.Ctx, &entity.SupplierEnterprise{SupplierID: fornA, EnterpriseCode: b.Code}); err == nil {
		t.Fatal("a A gravou vínculo em nome da B")
	}
	if _, err := repo.AddEnterprise(a.Ctx, &entity.SupplierEnterprise{SupplierID: fornA, EnterpriseCode: a.Code, PurchasePriceTableID: &tabelaB}); err == nil {
		t.Fatal("o vínculo da A aceitou a tabela de preço de compra da B")
	}
	vinculo, err := repo.AddEnterprise(a.Ctx, &entity.SupplierEnterprise{SupplierID: fornA, EnterpriseCode: a.Code, IsActive: true})
	if err != nil {
		t.Fatalf("a A não vinculou o próprio fornecedor: %v", err)
	}

	if _, err := repo.UpdateEnterprise(b.Ctx, &entity.SupplierEnterprise{ID: vinculo.ID, IsActive: false}); err == nil {
		t.Fatal("empresa B alterou o vínculo da A")
	}
	if _, err := repo.UpdateEnterprise(a.Ctx, &entity.SupplierEnterprise{ID: vinculo.ID, PurchasePriceTableID: &tabelaB, IsActive: true}); err == nil {
		t.Fatal("a alteração do vínculo da A aceitou a tabela de preço da B")
	}
	if err := repo.DeleteEnterprise(b.Ctx, vinculo.ID); err == nil {
		t.Fatal("empresa B excluiu o vínculo da A")
	}
	var ativo bool
	if err := pool.QueryRow(base, `SELECT is_active FROM supplier_enterprises WHERE id=$1`, vinculo.ID).Scan(&ativo); err != nil || !ativo {
		t.Fatalf("o vínculo da A foi alterado ou removido pela B (ativo=%v, %v)", ativo, err)
	}
	if _, err := repo.UpdateEnterprise(a.Ctx, &entity.SupplierEnterprise{ID: vinculo.ID, IsActive: false}); err != nil {
		t.Fatalf("a A não alterou o próprio vínculo: %v", err)
	}
	if err := repo.DeleteEnterprise(a.Ctx, vinculo.ID); err != nil {
		t.Fatalf("a A não excluiu o próprio vínculo: %v", err)
	}
}
