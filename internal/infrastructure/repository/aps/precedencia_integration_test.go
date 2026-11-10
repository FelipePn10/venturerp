//go:build integration

package aps_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Roteiro sem rede de precedência (o comum): as etapas seguem a SEQUÊNCIA,
// como na execução. Antes o APS não tinha aresta nenhuma e punha todas as
// etapas no mesmo instante. Etapa concluída sai da rede; a em andamento entra
// com o que falta, sem nova preparação.
func TestAPSSemRedeSegueASequencia(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM production_order_operations WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	var of int64
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,end_date,created_by,enterprise_id)
		VALUES($1,$2,1,'IN_PROGRESS',$3::date,$3::date,$4,$5) RETURNING id`, testutil.UniqueCode(), item, time.Now(), actor, a.ID).Scan(&of); err != nil {
		t.Fatal(err)
	}
	etapa := func(seq int, status string, previsto, prep, real string) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_order_operations(production_order_id,sequence,operation_name,status,planned_hours,setup_hours,actual_hours,enterprise_id)
			VALUES($1,$2,'E',$3,$4::numeric,$5::numeric,$6::numeric,$7) RETURNING id`, of, seq, status, previsto, prep, real, a.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	etapa(5, "DONE", "1", "0", "1")
	e10 := etapa(10, "IN_PROGRESS", "4", "1", "2")
	e20 := etapa(20, "PENDING", "2", "0.5", "0")
	e30 := etapa(30, "PENDING", "1", "0", "0")

	repo := apsrepo.New(sqlc.New(pool), pool)
	arestas, err := repo.GetOrderOperationEdges(a.Ctx, of)
	if err != nil {
		t.Fatal(err)
	}
	quer := map[[2]int64]bool{{e10, e20}: true, {e20, e30}: true}
	if len(arestas) != 2 {
		t.Fatalf("sem rede, 3 etapas abertas em sequência = 2 arestas; vieram %+v", arestas)
	}
	for _, e := range arestas {
		if !quer[[2]int64{e.PredecessorID, e.SuccessorID}] {
			t.Fatalf("aresta fora da sequência: %+v", e)
		}
	}
	ops, err := repo.GetOrderOperations(a.Ctx, of)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 3 || ops[0].ID != e10 || ops[0].PlannedHours != 3 || ops[0].SetupHours != 0 {
		t.Fatalf("etapa em andamento entra com o que falta (4 + 1 − 2 = 3 h, sem preparação): %+v", ops)
	}
	if ops[1].PlannedHours != 2 || ops[1].SetupHours != 0.5 {
		t.Fatalf("etapa pendente mantém previsto e preparação: %+v", ops[1])
	}
}
