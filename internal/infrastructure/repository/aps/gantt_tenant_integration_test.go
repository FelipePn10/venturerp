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

// O APS e o quadro mensal enxergam só as ordens e a carga da empresa da
// sessão; a carga de cenário de simulação não entra no quadro.
func TestAPSEGanttIsoladosPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM capacity_requirements WHERE plan_code IN (SELECT code FROM production_plans WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	itemA, itemB := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, itemA, actor)
	testutil.SeedItem(t, pool, b.Ctx, itemB, actor)
	dia := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	of := func(e testutil.Empresa, item int64) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,end_date,created_by,enterprise_id)
			VALUES($1,$2,1,'OPEN',$3::date,$3::date,$4,$5) RETURNING id`, testutil.UniqueCode(), item, dia, actor, e.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	ofA, ofB := of(a, itemA), of(b, itemB)
	repo := apsrepo.New(sqlc.New(pool), pool)

	abertas, err := repo.GetOpenProductionOrders(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range abertas {
		if o.ID == ofB {
			t.Fatal("o APS da empresa A percorre a OF da empresa B")
		}
	}
	gantt := repo
	barras, err := gantt.ListFallbackBars(a.Ctx, dia.AddDate(0, 0, -1), dia.AddDate(0, 0, 2))
	if err != nil {
		t.Fatal(err)
	}
	viuA := false
	for _, br := range barras {
		if br.ProductionOrderID == ofB {
			t.Fatal("o quadro da empresa A mostra a OF da empresa B")
		}
		if br.ProductionOrderID == ofA {
			viuA = true
			if br.ItemBusinessCode == "" {
				t.Fatal("barra sem o código de negócio do item")
			}
		}
	}
	if !viuA {
		t.Fatal("o quadro da empresa A não mostra a própria OF")
	}

	// Carga: plano real da A conta; cenário da A e plano da B não.
	var wc int64
	if err := pool.QueryRow(base, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) VALUES($1,'Gantt WC',(enum_range(NULL::machine_type_enum))[1],$2,$3) RETURNING id`, testutil.UniqueCode(), actor, a.ID).Scan(&wc); err != nil {
		t.Fatal(err)
	}
	real, cenario, daB := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id,is_scenario) VALUES($1,'Real',$4,$5,false),($2,'Cenário',$4,$5,true),($3,'Da B',$4,$6,false)`, real, cenario, daB, actor, a.ID, b.ID)
	for _, p := range []int64{real, cenario, daB} {
		testutil.Exec(t, pool, `INSERT INTO capacity_requirements(plan_code,work_center_id,req_date,required_hours,available_hours) VALUES($1,$2,$3::date,5,8)`, p, wc, dia)
	}
	carga, err := gantt.ListResourceLoad(a.Ctx, dia, dia.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	total := 0.0
	for _, c := range carga {
		if c.WorkCenterID == wc {
			total += c.RequiredHours
		}
	}
	if total != 5 {
		t.Fatalf("carga do centro = %v h, esperado 5 (só o plano real da empresa A)", total)
	}
}
