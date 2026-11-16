//go:build integration

package sales_order_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A segunda confirmação do MESMO cliente espera a primeira terminar; outro
// cliente, ou o mesmo código em outra empresa, segue livre.
func TestTravaDeCreditoPorCliente(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	cliente := testutil.UniqueCode()

	tx1, err := pool.Begin(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback(a.Ctx)
	if err := salesorder.NewSalesOrderRepositorySQLC(sqlc.New(tx1)).LockCustomerCredit(a.Ctx, cliente); err != nil {
		t.Fatal(err)
	}

	tenta := func(e testutil.Empresa, codigo int64) error {
		t.Helper()
		tx, err := pool.Begin(e.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(e.Ctx)
		if _, err := tx.Exec(e.Ctx, `SET LOCAL lock_timeout = '300ms'`); err != nil {
			t.Fatal(err)
		}
		return salesorder.NewSalesOrderRepositorySQLC(sqlc.New(tx)).LockCustomerCredit(e.Ctx, codigo)
	}
	if err := tenta(a, cliente); err == nil {
		t.Fatal("a segunda análise do mesmo cliente não esperou a primeira")
	}
	if err := tenta(a, cliente+1); err != nil {
		t.Fatalf("outro cliente ficou travado: %v", err)
	}
	if err := tenta(b, cliente); err != nil {
		t.Fatalf("o mesmo código em outra empresa ficou travado: %v", err)
	}
	tx1.Rollback(a.Ctx)
	if err := tenta(a, cliente); err != nil {
		t.Fatalf("a trava não saiu com o fim da transação: %v", err)
	}
}
