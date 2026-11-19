//go:build integration

package sales_order_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Pedido que entra por importação ou carga direta (sem passar pelo contador)
// não faz o próximo pedido da tela repetir o número — antes dava "já existe um
// registro" e o pedido não saía. O contador de cada empresa é só dela.
func TestNumeroDoPedidoDeVendaPulaOQueEntrouPorFora(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE enterprise_code=$1`, e.Code)
			testutil.Exec(t, pool, `DELETE FROM sales_order_sequences WHERE enterprise_code=$1`, e.Code)
		}
	})
	repo := salesorder.NewSalesOrderRepositorySQLC(sqlc.New(pool))
	if n, err := repo.NextOrderNumber(a.Ctx, a.Code); err != nil || n != 1 {
		t.Fatalf("primeiro número da A: %d %v", n, err)
	}
	// Carga direta: os pedidos 2 a 40 entram sem passar pelo contador.
	testutil.Exec(t, pool, `INSERT INTO sales_orders(order_number,enterprise_code,created_by) SELECT g,$1,$2 FROM generate_series(2,40) g`, a.Code, ator)
	if n, err := repo.NextOrderNumber(a.Ctx, a.Code); err != nil || n != 41 {
		t.Fatalf("depois da carga direta o próximo da A tem de ser 41; veio %d %v", n, err)
	}
	// A empresa B não herda a numeração da A.
	if n, err := repo.NextOrderNumber(b.Ctx, b.Code); err != nil || n != 1 {
		t.Fatalf("primeiro número da B: %d %v", n, err)
	}
	if n, err := repo.NextOrderNumber(a.Ctx, a.Code); err != nil || n != 42 {
		t.Fatalf("a A segue a própria sequência: %d %v", n, err)
	}
}
