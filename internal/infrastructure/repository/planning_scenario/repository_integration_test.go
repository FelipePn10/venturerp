//go:build integration

package planning_scenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	apsdomain "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	capacity "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine_capacity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	mrpreport "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_report"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_scenario"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O cenário é da empresa, calcula num plano oculto e não toca na operação:
// não aparece na lista de planos, a reserva de máquina dele não ocupa a
// máquina do plano real (MRP e APS), e ele ignora as reservas não firmadas do
// próprio plano base. Os ajustes de capacidade e estoque valem só para ele.
func TestCenarioIsoladoDoPlanoReal(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM planning_scenarios WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_machine_allocation_slots WHERE suggestion_code IN (SELECT code FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_item_profiles WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM independent_demands WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machines WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	plano, outroPlano := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'Base',$3,$4),($2,'Outro',$3,$4)`, plano, outroPlano, actor, a.ID)
	tipoCode := testutil.UniqueCode()
	var tipo, maquina int64
	if err := pool.QueryRow(base, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id,capacity_hours,is_active) VALUES($1,'Corte','CUT',$2,$3,8,true) RETURNING id`, tipoCode, actor, a.ID).Scan(&tipo); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO machines(code,name,machine_type_code,created_by,enterprise_id,is_active,efficiency_rate,available_hours_per_day) VALUES($1,'Serra',$1,$2,$3,true,1,8) RETURNING id`, tipoCode, actor, a.ID).Scan(&maquina); err != nil {
		t.Fatal(err)
	}

	repo := planning_scenario.New(pool)
	cen, err := repo.Create(a.Ctx, "Simulação", "", plano, actor)
	if err != nil {
		t.Fatal(err)
	}
	if cen.ScenarioPlanCode < 1_000_000_000 {
		t.Fatalf("plano oculto %d fora da faixa reservada", cen.ScenarioPlanCode)
	}
	var oculto bool
	if err := pool.QueryRow(base, `SELECT is_scenario FROM production_plans WHERE code=$1`, cen.ScenarioPlanCode).Scan(&oculto); err != nil || !oculto {
		t.Fatalf("plano do cenário não é oculto: %v %v", oculto, err)
	}
	planos, err := sqlc.New(pool).ListProductionPlans(base, &a.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range planos {
		if p.Code == cen.ScenarioPlanCode {
			t.Fatal("o plano oculto aparece na lista de planos")
		}
	}

	// Outra empresa não enxerga nem mexe no cenário.
	if _, err := repo.Get(b.Ctx, cen.ID); err == nil {
		t.Fatal("empresa B leu o cenário da A")
	}
	if lista, err := repo.List(b.Ctx); err != nil || len(lista) != 0 {
		t.Fatalf("empresa B lista cenários da A: %+v (%v)", lista, err)
	}
	if err := repo.Delete(b.Ctx, cen.ID); err == nil {
		t.Fatal("empresa B descartou o cenário da A")
	}
	if _, err := repo.AddChange(b.Ctx, cen.ID, planning_uc.ScenarioChange{Kind: planning_uc.ChangeStock, ItemCode: &item}, actor); err == nil {
		t.Fatal("empresa B ajustou o cenário da A")
	}

	q := func(v int64) *decimal.Decimal { d := decimal.NewFromInt(v); return &d }
	quatro := decimal.NewFromInt(4)
	segunda := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	terca := segunda.AddDate(0, 0, 1)
	sete := 7
	for _, c := range []planning_uc.ScenarioChange{
		{Kind: planning_uc.ChangeStock, ItemCode: &item, Quantity: q(100)},
		{Kind: planning_uc.ChangeLeadTime, ItemCode: &item, LeadTimeDays: &sete},
		{Kind: planning_uc.ChangeDemand, DemandRefKind: planning_uc.DemandRefNew, ItemCode: &item, Quantity: q(12), NeedDate: &segunda},
		{Kind: planning_uc.ChangeCapacity, MachineID: &maquina, HoursPerDay: &quatro},
		{Kind: planning_uc.ChangeCapacity, MachineID: &maquina, StopFrom: &terca},
	} {
		if _, err := repo.AddChange(a.Ctx, cen.ID, c, actor); err != nil {
			t.Fatal(err)
		}
	}
	adj, err := repo.ScenarioAdjustments(a.Ctx, cen.ScenarioPlanCode)
	if err != nil || !adj.StockDelta[item].Equal(decimal.NewFromFloat(100)) || adj.LeadTimeDays[item] != 7 || len(adj.Demands) != 1 {
		t.Fatalf("ajustes do cenário: %+v (%v)", adj, err)
	}
	if real, err := repo.ScenarioAdjustments(a.Ctx, plano); err != nil || len(real.Demands) != 0 || len(real.StockDelta) != 0 {
		t.Fatalf("plano real com ajuste de cenário: %+v (%v)", real, err)
	}

	// Capacidade do CRP: 4 h no cenário, 8 h no real; a parada zera o dia.
	for _, c := range []struct {
		plan int64
		dia  time.Time
		want float64
	}{{cen.ScenarioPlanCode, segunda, 4}, {plano, segunda, 8}, {cen.ScenarioPlanCode, terca, 0}, {plano, terca, 8}} {
		dia := c.dia
		h, err := capacity.HoursForPlan(a.Ctx, pool, tipo, &dia, c.plan)
		if err != nil || h != c.want {
			t.Fatalf("plano %d em %s: %v h (%v), esperado %v", c.plan, dia.Format("02/01"), h, err, c.want)
		}
	}

	// Reservas de máquina: cenário, plano base (não firmada) e outro plano real.
	reserva := func(plan int64, ini, fim int) {
		var sug int64
		if err := pool.QueryRow(base, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,start_date,order_type,enterprise_id) VALUES($1,$2,1,$3::date,$3::date,'FABRICACAO',$4) RETURNING code`,
			plan, item, segunda, a.ID).Scan(&sug); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `INSERT INTO mrp_machine_allocations(suggestion_code,enterprise_id,machine_id,production_minutes,schedule_date) VALUES($1,$2,$3,60,$4::date)`, sug, a.ID, maquina, segunda)
		testutil.Exec(t, pool, `INSERT INTO mrp_machine_allocation_slots(machine_id,suggestion_code,starts_at,ends_at) VALUES($1,$2,$3,$4)`,
			maquina, sug, segunda.Add(time.Duration(ini)*time.Hour), segunda.Add(time.Duration(fim)*time.Hour))
	}
	reserva(cen.ScenarioPlanCode, 8, 9)
	reserva(plano, 9, 10)
	reserva(outroPlano, 10, 11)
	inicios := func(ws []time.Time) map[int]bool {
		out := map[int]bool{}
		for _, w := range ws {
			if w.Day() == segunda.Day() {
				out[w.Hour()] = true
			}
		}
		return out
	}
	mrp := mrprepo.NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	ocupado := func(plan int64) map[int]bool {
		t.Helper()
		_, busy, err := mrp.MachinePlanningWindows(a.Ctx, maquina, plan, segunda, terca)
		if err != nil {
			t.Fatal(err)
		}
		ts := []time.Time{}
		for _, w := range busy {
			ts = append(ts, w.Start)
		}
		return inicios(ts)
	}
	if o := ocupado(plano); o[8] || !o[10] {
		t.Fatalf("plano real: ocupado %v — a reserva do cenário (08h) não pode contar; a do outro plano (10h) conta", o)
	}
	if o := ocupado(cen.ScenarioPlanCode); o[9] || !o[10] {
		t.Fatalf("cenário: ocupado %v — a reserva não firmada do plano base (09h) não pode contar; a do outro plano (10h) conta", o)
	}
	janelas, busyCen, err := mrp.MachinePlanningWindows(a.Ctx, maquina, cen.ScenarioPlanCode, segunda, segunda)
	if err != nil || len(janelas) != 1 || janelas[0].End.Sub(janelas[0].Start) != 4*time.Hour {
		t.Fatalf("janela do cenário: %+v (%v), esperado um turno de 4 h", janelas, err)
	}
	_ = busyCen
	_, busyTerca, err := mrp.MachinePlanningWindows(a.Ctx, maquina, cen.ScenarioPlanCode, terca, terca)
	if err != nil {
		t.Fatal(err)
	}
	parada := false
	for _, w := range busyTerca {
		if !w.Start.After(terca) && !w.End.Before(terca.Add(24*time.Hour)) {
			parada = true
		}
	}
	if !parada {
		t.Fatalf("a parada do cenário não bloqueou a terça: %+v", busyTerca)
	}

	aps := apsrepo.New(sqlc.New(pool), pool).(apsdomain.SelectionRepository)
	paradas, err := aps.ListMachineDowntimeWindows(a.Ctx, maquina, segunda, terca)
	if err != nil {
		t.Fatal(err)
	}
	// O APS devolve a reserva (hora de parede da fábrica) no fuso da empresa.
	brt, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	ts := []time.Time{}
	for _, w := range paradas {
		l := w.Start.In(brt)
		ts = append(ts, time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), 0, 0, 0, time.UTC))
	}
	if o := inicios(ts); o[8] || !o[9] || !o[10] {
		t.Fatalf("APS: ocupado %v — reserva de cenário (08h) não ocupa a programação real", o)
	}

	// Cobertura do cenário soma a entrada simulada ao estoque real.
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,enterprise_id) VALUES($1,1,10,$2)`, item, a.ID)
	testutil.Exec(t, pool, `INSERT INTO mrp_item_profiles(item_code,plan_code,calculation_date,demand,need_date,enterprise_id) VALUES($1,$2,$3::date,5,$3::date,$4),($1,$5,$3::date,5,$3::date,$4)`,
		item, cen.ScenarioPlanCode, segunda, a.ID, plano)
	cobertura := func(plan int64) decimal.Decimal {
		rows, err := mrpreport.New(pool).Coverage(a.Ctx, mrp_report_uc.Filter{PlanCode: &plan})
		if err != nil || len(rows) != 1 {
			t.Fatalf("cobertura do plano %d: %+v (%v)", plan, rows, err)
		}
		return rows[0].Stock
	}
	if s := cobertura(cen.ScenarioPlanCode); !s.Equal(decimal.NewFromInt(110)) {
		t.Fatalf("estoque do cenário = %s, esperado 110", s)
	}
	if s := cobertura(plano); !s.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("estoque do plano real = %s, esperado 10", s)
	}

	// Promover cria a demanda nova na empresa A; não promove duas vezes.
	res, err := repo.Promote(a.Ctx, cen.ID, actor)
	if err != nil || res.Created != 1 {
		t.Fatalf("promoção: %+v (%v)", res, err)
	}
	var demandas int
	if err := pool.QueryRow(base, `SELECT COUNT(*) FROM independent_demands WHERE enterprise_id=$1 AND item_code=$2 AND quantity=12`, a.ID, item).Scan(&demandas); err != nil || demandas != 1 {
		t.Fatalf("demanda promovida: %d (%v)", demandas, err)
	}
	if _, err := repo.Promote(a.Ctx, cen.ID, actor); err == nil {
		t.Fatal("promoveu duas vezes")
	}

	// Descartar apaga o cenário, o plano oculto e o que ele calculou.
	if err := repo.Delete(a.Ctx, cen.ID); err != nil {
		t.Fatal(err)
	}
	var sobras int
	if err := pool.QueryRow(base, `SELECT (SELECT COUNT(*) FROM production_plans WHERE code=$1) + (SELECT COUNT(*) FROM mrp_planned_suggestions WHERE plan_code=$1) + (SELECT COUNT(*) FROM mrp_item_profiles WHERE plan_code=$1)`,
		cen.ScenarioPlanCode).Scan(&sobras); err != nil || sobras != 0 {
		t.Fatalf("sobrou %d registro(s) do cenário descartado (%v)", sobras, err)
	}
}
