//go:build integration

package production_order_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Cancelar a OF fecha o que ficou pendurado nela (OS sem remessa e lote de
// inspeção aberto) e, quando não pode, diz por quê — antes respondia "registro
// não encontrado" com a OF ali. Encerrar fora de "concluída" também explica.
func TestCancelarOFFechaPendenciasEExplicaARecusa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM quality_inspection_lots WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM third_party_service_order_history WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM third_party_service_movements WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM third_party_service_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_appointments WHERE production_order_id IN (SELECT id FROM production_orders WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM route_operations WHERE route_id IN (SELECT id FROM manufacturing_routes WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM operations WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	novaOF := func(status string) int64 {
		var id int64
		must(pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,origin_type,created_by,enterprise_id)
			VALUES($1,$2,10,$3,'MANUAL',$4,$5) RETURNING id`, testutil.UniqueCode(), item, status, actor, a.ID).Scan(&id))
		return id
	}
	var operacao, roteiro, etapa int64
	must(pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id) VALUES($1,'Zincagem','TERCEIROS',$2,$3) RETURNING id`, testutil.UniqueCode(), actor, a.ID).Scan(&operacao))
	must(pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,is_active,created_by,enterprise_id) VALUES($1,$2,1,TRUE,TRUE,$3,$4) RETURNING id`, testutil.UniqueCode(), item, actor, a.ID).Scan(&roteiro))
	must(pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id,is_active) VALUES($1,10,$2,TRUE) RETURNING id`, roteiro, operacao).Scan(&etapa))
	hoje := time.Now()
	novaOS := func(of int64) int64 {
		var id int64
		must(pool.QueryRow(base, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,uom,quantity,start_date,due_date,status,created_by)
			VALUES($1,$2,$3,$4,$5,$6,'UN',10,$7::date,$7::date,'FIRM',$8) RETURNING id`, testutil.UniqueCode()%1_000_000, a.ID, of, etapa, operacao, item, hoje, actor).Scan(&id))
		return id
	}
	repo := production_order.NewProductionOrderRepositoryPGX(pool)

	// 1) OF aberta com OS firme e lote de inspeção aberto: tudo cancela junto.
	of := novaOF("OPEN")
	os := novaOS(of)
	var lote int64
	must(pool.QueryRow(base, `INSERT INTO quality_inspection_lots(enterprise_id,numero,origem,item_code,quantidade,production_order_id)
		VALUES($1,$2,'PROCESSO',$3,10,$4) RETURNING id`, a.ID, testutil.UniqueCode()%1_000_000, item, of).Scan(&lote))
	if _, err := repo.Cancel(b.Ctx, of); err == nil {
		t.Fatal("a empresa B cancelou a OF da empresa A")
	}
	_, err := repo.Cancel(a.Ctx, of)
	must(err)
	var stOS, stLote string
	must(pool.QueryRow(base, `SELECT status FROM third_party_service_orders WHERE id=$1`, os).Scan(&stOS))
	must(pool.QueryRow(base, `SELECT status FROM quality_inspection_lots WHERE id=$1`, lote).Scan(&stLote))
	if stOS != "CANCELLED" || stLote != "CANCELADO" {
		t.Fatalf("OS=%s lote=%s: a OF cancelada deixou pendência aberta", stOS, stLote)
	}
	var historico int
	must(pool.QueryRow(base, `SELECT COUNT(*) FROM third_party_service_order_history WHERE service_order_id=$1 AND previous_status='FIRM' AND new_status='CANCELLED'`, os).Scan(&historico))
	if historico != 1 {
		t.Fatalf("o cancelamento da OS não ficou no histórico: %d", historico)
	}

	espera := func(err error, trecho string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), trecho) {
			t.Fatalf("quer recusa com %q, veio %v", trecho, err)
		}
	}
	// 2) Peças com o terceiro: recusa.
	of2 := novaOF("IN_PROGRESS")
	os2 := novaOS(of2)
	_, err = pool.Exec(base, `INSERT INTO third_party_service_movements(enterprise_id,service_order_id,movement_type,quantity,occurred_at,created_by) VALUES($1,$2,'REMITTANCE',10,NOW(),$3)`, a.ID, os2, actor)
	must(err)
	_, err = repo.Cancel(a.Ctx, of2)
	espera(err, "terceiro")
	// 3) Com apontamento: recusa explicando.
	of3 := novaOF("IN_PROGRESS")
	_, err = pool.Exec(base, `INSERT INTO production_appointments(production_order_id,appointment_date,produced_qty,scrapped_qty,created_by) VALUES($1,CURRENT_DATE,1,0,$2)`, of3, actor)
	must(err)
	_, err = repo.Cancel(a.Ctx, of3)
	espera(err, "apontamento")
	// 4) Encerrar o que não está concluído explica a situação.
	_, err = repo.Close(a.Ctx, of3)
	espera(err, "em produção")
	_, err = repo.Close(a.Ctx, 999999999)
	espera(err, "não encontrada")
}
