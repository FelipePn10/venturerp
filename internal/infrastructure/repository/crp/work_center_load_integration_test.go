//go:build integration

package crp_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	crprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/crp"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A carga de um centro soma só os planos reais da empresa da sessão: plano de
// cenário de simulação e plano de outra empresa ficam de fora.
func TestCargaDoCentroSoPlanosReaisDaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM capacity_requirements WHERE plan_code IN (SELECT code FROM production_plans WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id = ANY($1)`, ids)
	})
	var wc int64
	if err := pool.QueryRow(base, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) VALUES($1,'CRP WC',(enum_range(NULL::machine_type_enum))[1],$2,$3) RETURNING id`,
		testutil.UniqueCode(), actor, a.ID).Scan(&wc); err != nil {
		t.Fatal(err)
	}
	real, cenario, daB := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id,is_scenario) VALUES($1,'Real',$4,$5,false),($2,'Cenário',$4,$5,true),($3,'Da B',$4,$6,false)`,
		real, cenario, daB, actor, a.ID, b.ID)
	dia := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	for _, p := range []int64{real, cenario, daB} {
		testutil.Exec(t, pool, `INSERT INTO capacity_requirements(plan_code,work_center_id,req_date,required_hours,available_hours) VALUES($1,$2,$3::date,5,8)`, p, wc, dia)
	}
	repo := crprepo.New(sqlc.New(pool), pool)
	carga, err := repo.ListByWorkCenter(a.Ctx, wc, dia.AddDate(0, 0, -1), dia.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(carga) != 1 || carga[0].PlanCode != real {
		t.Fatalf("carga do centro = %+v; esperado só o plano real %d", carga, real)
	}
	if outra, err := repo.ListByWorkCenter(b.Ctx, wc, dia.AddDate(0, 0, -1), dia.AddDate(0, 0, 1)); err != nil || len(outra) != 1 || outra[0].PlanCode != daB {
		t.Fatalf("empresa B: %+v (%v); esperado só o plano %d dela", outra, err, daB)
	}
}
