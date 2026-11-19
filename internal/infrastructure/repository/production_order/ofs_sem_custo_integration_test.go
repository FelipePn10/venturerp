//go:build integration

package production_order_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestOFsSemCustoEntramNaFila: OF que se moveu (horas de etapa, apontamento
// ou estoque) e nunca teve custo entra na fila, com a empresa dela; OF sem
// movimento, cancelada ou já apurada fica de fora. A fila corre todas as
// empresas, cada OF vai com a sua — a apuração acontece na sessão dela.
func TestOFsSemCustoEntramNaFila(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	itA, itB := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, itA)
	testutil.NovosItens(t, pool, b.ID, actor, itB)
	of := func(emp, item int64, status string) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id)
			VALUES($1,$2,10,$3,$4,$5) RETURNING id`, testutil.UniqueCode()%1_000_000, item, status, actor, emp).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM production_orders WHERE id = $1`, id) })
		return id
	}
	horas := func(emp, ordem int64) {
		t.Helper()
		testutil.Exec(t, pool, `INSERT INTO production_order_operations(production_order_id,sequence,operation_name,actual_hours,enterprise_id) VALUES($1,10,'Corte',1.5,$2)`, ordem, emp)
	}
	comHorasA, comHorasB := of(a.ID, itA, "COMPLETED"), of(b.ID, itB, "IN_PROGRESS")
	horas(a.ID, comHorasA)
	horas(b.ID, comHorasB)
	semMovimento := of(a.ID, itA, "OPEN")
	cancelada := of(a.ID, itA, "CANCELLED")
	horas(a.ID, cancelada)
	apurada := of(a.ID, itA, "COMPLETED")
	horas(a.ID, apurada)
	testutil.Exec(t, pool, `INSERT INTO production_order_costs(production_order_id, produced_qty, settled_by) VALUES($1, 10, $2)`, apurada, actor)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM production_order_costs WHERE production_order_id = $1`, apurada)
	})

	fila, err := productionrepo.NewProductionOrderRepositoryPGX(pool).OFsSemCusto(base, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	naFila := map[int64]entity.OFSemCusto{}
	for _, f := range fila {
		naFila[f.ID] = f
	}
	if f, ok := naFila[comHorasA]; !ok || f.EnterpriseID != a.ID || f.EnterpriseCode != a.Code {
		t.Fatalf("OF da A com horas e sem custo deveria estar na fila com a empresa A: %+v", f)
	}
	if f, ok := naFila[comHorasB]; !ok || f.EnterpriseID != b.ID {
		t.Fatalf("OF da B deveria estar na fila com a empresa B: %+v", f)
	}
	for nome, id := range map[string]int64{"sem movimento": semMovimento, "cancelada": cancelada, "já apurada": apurada} {
		if _, ok := naFila[id]; ok {
			t.Errorf("OF %s entrou na fila", nome)
		}
	}
}
