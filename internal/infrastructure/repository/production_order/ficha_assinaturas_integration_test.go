//go:build integration

package production_order_test

import (
	"context"
	"testing"

	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A ficha traz o histórico das etapas (com o motivo da pausa), a instrução do
// roteiro e as assinaturas pelo que o sistema registrou; a empresa B não lê.
func TestFichaDaOrdemHistoricoEAssinaturas(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	var of, op, rota, roteiro, operacao, lote int64
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id) VALUES($1,'Solda','INTERNA',$2,$3) RETURNING id`, testutil.UniqueCode(), ator, a.ID).Scan(&operacao))
	must(pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) VALUES($1,$2,1,FALSE,$3,$4) RETURNING id`, testutil.UniqueCode(), item, ator, a.ID).Scan(&roteiro))
	must(pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id,notes) VALUES($1,10,$2,'Conferir o esquadro') RETURNING id`, roteiro, operacao).Scan(&rota))
	must(pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,priority,notes,created_by,enterprise_id,route_id)
		VALUES($1,$2,2,'IN_PROGRESS','HIGH','Rebarbar',$3,$4,$5) RETURNING id`, testutil.UniqueCode()%1_000_000, item, ator, a.ID, roteiro).Scan(&of))
	must(pool.QueryRow(base, `INSERT INTO production_order_operations(production_order_id,route_operation_id,sequence,operation_name,status,enterprise_id)
		VALUES($1,$2,10,'Solda','PENDING',$3) RETURNING id`, of, rota, a.ID).Scan(&op))
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM quality_lot_decisions WHERE lot_id=$1`, lote)
		pool.Exec(base, `DELETE FROM quality_inspection_lots WHERE id=$1`, lote)
		pool.Exec(base, `DELETE FROM production_order_operations WHERE id=$1`, op)
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, of)
		pool.Exec(base, `DELETE FROM route_operations WHERE id=$1`, rota)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operacao)
	})
	// O apontamento grava o autor na sessão e a trilha vem do gatilho.
	for _, passo := range []struct{ status, nota string }{{"IN_PROGRESS", "· Em andamento"}, {"PAUSED", "· Pausada: almoço"}, {"IN_PROGRESS", "· Em andamento"}, {"DONE", "· Concluída"}} {
		tx, err := pool.Begin(base)
		must(err)
		_, err = tx.Exec(base, `SELECT set_config('venture.execution_actor', $1, true)`, ator.String())
		must(err)
		_, err = tx.Exec(base, `UPDATE production_order_operations SET status=$2, notes=concat_ws(E'\n', notes, '2026-11-23 10:00:00 UTC '||$3) WHERE id=$1`, op, passo.status, passo.nota)
		must(err)
		must(tx.Commit(base))
	}
	must(pool.QueryRow(base, `INSERT INTO quality_inspection_lots(enterprise_id,numero,origem,item_code,quantidade,production_order_id,status,created_by)
		VALUES($1,$2,'FINAL',$3,2,$4,'DECIDIDO',$5) RETURNING id`, a.ID, testutil.UniqueCode()%1_000_000, item, of, ator).Scan(&lote))
	_, err := pool.Exec(base, `INSERT INTO quality_lot_decisions(lot_id,enterprise_id,decisao,qtd_aceita,decided_at,decided_by) VALUES($1,$2,'ACEITO',2,NOW(),$3)`, lote, a.ID, ator)
	must(err)

	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	if _, err := repo.FichaDaOrdem(b.Ctx, of); err == nil {
		t.Fatal("a empresa B leu a ficha da OF da A")
	}
	f, err := repo.FichaDaOrdem(a.Ctx, of)
	must(err)
	e := f.Etapas[0]
	if e.Instrucao != "Conferir o esquadro" || len(e.Eventos) != 4 || e.Eventos[1].Status != "PAUSED" || e.Eventos[1].Motivo != "almoço" || e.Eventos[3].Quem == "" {
		t.Fatalf("etapa: instrução=%q eventos=%+v", e.Instrucao, e.Eventos)
	}
	if f.Emissao.Quando == nil || f.Emissao.Nome == "" {
		t.Fatalf("emissão sem autor/data: %+v", f.Emissao)
	}
	if f.Execucao.Quando != nil {
		t.Fatal("execução só assina com a ordem concluída")
	}
	if f.InspecaoFinal == nil || f.InspecaoFinal.Decisao != "ACEITO" || f.Qualidade.Quando == nil {
		t.Fatalf("qualidade: %+v %+v", f.InspecaoFinal, f.Qualidade)
	}
}
