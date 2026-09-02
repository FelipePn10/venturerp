//go:build integration

package customer_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestCommercialPricingIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	codeA := int64(1_650_000_000 + testutil.UniqueCode()%10_000_000)
	codeB := codeA + 1
	var enterpriseA, enterpriseB, tableA, tableB int64
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Pricing A') RETURNING id`, codeA).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Pricing B') RETURNING id`, codeB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO sales_tables(code,description,enterprise_id) VALUES($1,'Tabela A',$2) RETURNING id`, codeA, enterpriseA).Scan(&tableA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO sales_tables(code,description,enterprise_id) VALUES($1,'Tabela B',$2) RETURNING id`, codeB, enterpriseB).Scan(&tableB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM sales_table_price_history WHERE enterprise_id=ANY($1)`, []int64{enterpriseA, enterpriseB})
		_, _ = pool.Exec(base, `DELETE FROM sales_table_prices WHERE sales_table_id=ANY($1)`, []int64{tableA, tableB})
		_, _ = pool.Exec(base, `DELETE FROM sales_tables WHERE id=ANY($1)`, []int64{tableA, tableB})
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=ANY($1)`, []int64{enterpriseA, enterpriseB})
	})

	repo := customerpg.New(sqlc.New(pool), pool)
	ctxA := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseA, EnterpriseCode: codeA})
	ctxB := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseB, EnterpriseCode: codeB})
	created, err := repo.CreateSalesTablePrice(ctxA, &entity.SalesTablePrice{SalesTableID: tableA, ItemCode: "10001", Price: 125, Situation: entity.PriceSituationAtivo})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetSalesTablePriceByID(ctxB, created.ID); err == nil {
		t.Fatal("tenant B leu preço do tenant A")
	}
	if _, err = repo.CreateSalesTablePrice(ctxB, &entity.SalesTablePrice{SalesTableID: tableA, ItemCode: "10002", Price: 130, Situation: entity.PriceSituationAtivo}); err == nil {
		t.Fatal("tenant B incluiu preço na tabela do tenant A")
	}
	if err = repo.DeleteSalesTablePrice(ctxB, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetSalesTablePriceByID(ctxA, created.ID); err != nil {
		t.Fatalf("exclusão cruzada removeu preço do tenant A: %v", err)
	}
}
