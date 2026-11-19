//go:build integration

package aps_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O sequenciamento lê a cadeia OF filha → OF pai da empresa: a OF da bucha
// aberta aparece como filha da OF do RN; a concluída não; outra empresa não
// enxerga nada.
func TestAPSLeACadeiaDaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	rn, bu := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, rn, bu)
	hoje := time.Now()
	planejada := func(item int64) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm)
			VALUES($1,$2,10,'PRODUCTION','SALES_ORDER',$3::date,$4,1,$5,TRUE) RETURNING id`, testutil.UniqueCode()%1_000_000, item, hoje, actor, a.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	rnPlan, buPlan, bu2Plan := planejada(rn), planejada(bu), planejada(bu)
	testutil.Exec(t, pool, `INSERT INTO production_chain_links(enterprise_id,child_planned_order_id,parent_planned_order_id,quantity,created_by)
		VALUES($1,$2,$3,10,$5),($1,$4,$3,10,$5)`, a.ID, buPlan, rnPlan, bu2Plan, actor)
	of := func(item, plan int64, status string) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id,planned_order_id)
			VALUES($1,$2,10,$3,$4::date,$5,$6,$7) RETURNING id`, testutil.UniqueCode()%1_000_000, item, status, hoje, actor, a.ID, plan).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	ofRN, ofBU := of(rn, rnPlan, "OPEN"), of(bu, buPlan, "RELEASED")
	of(bu, bu2Plan, "COMPLETED")

	cadeia := apsrepo.New(sqlc.New(pool), pool).(repository.CadeiaRepository)
	filhas, err := cadeia.OrdensFilhas(a.Ctx, []int64{ofRN, ofBU})
	if err != nil {
		t.Fatal(err)
	}
	if len(filhas) != 1 || len(filhas[ofRN]) != 1 || filhas[ofRN][0] != ofBU {
		t.Fatalf("filhas = %v, esperado só a OF %d (aberta) para a OF %d", filhas, ofBU, ofRN)
	}
	if outra, err := cadeia.OrdensFilhas(b.Ctx, []int64{ofRN}); err != nil || len(outra) != 0 {
		t.Fatalf("a outra empresa enxergou a cadeia: %v (%v)", outra, err)
	}
	var wc int64
	if err := pool.QueryRow(base, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) VALUES($1,'Cadeia WC',(enum_range(NULL::machine_type_enum))[1],$2,$3) RETURNING id`,
		testutil.UniqueCode(), actor, a.ID).Scan(&wc); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM production_sequences WHERE production_order_id=$1`, ofBU)
		testutil.Exec(t, pool, `DELETE FROM machine_types WHERE id=$1`, wc)
	})
	fim := hoje.Add(48 * time.Hour).Truncate(time.Second)
	testutil.Exec(t, pool, `INSERT INTO production_sequences(production_order_id,work_center_id,sequence_position,scheduled_start,scheduled_end,status,enterprise_id)
		VALUES($1,$2,10,$3,$4,'SCHEDULED',$5)`, ofBU, wc, hoje, fim, a.ID)
	fins, err := cadeia.FimProgramado(a.Ctx, []int64{ofBU})
	if err != nil || !fins[ofBU].Equal(fim) {
		t.Fatalf("fim programado da bucha = %v (%v), esperado %v", fins, err, fim)
	}
	if outra, err := cadeia.FimProgramado(b.Ctx, []int64{ofBU}); err != nil || len(outra) != 0 {
		t.Fatalf("a outra empresa leu a programação: %v (%v)", outra, err)
	}
}
