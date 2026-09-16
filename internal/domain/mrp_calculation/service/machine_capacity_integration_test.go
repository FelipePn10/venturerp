//go:build integration

package service

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc"
	productionuc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	machinesvc "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	crprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/crp"
	machinerepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine"
	capacity "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine_capacity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"math"
	"testing"
	"time"
)

func TestMachineProductivityMRPAndCRPIntegration(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := uuid.New()
	code := testutil.UniqueCode()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,name,email,password) VALUES($1,'Capacity test',$2,'x')`, actor, actor.String()+"@test.local")
	var enterprise int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Capacity test',$2) RETURNING id`, code%1000000000, actor).Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	q := sqlc.New(pool)
	machines := machinerepo.NewMachineRepositorySQLC(q, pool)
	mt, err := machines.CreateType(ctx, &machineentity.MachineType{Code: code, Name: "Cut", Type: types.MachineCut, IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	hours := 6.0
	eff := 0.8
	m, err := machines.Create(ctx, &machineentity.Machine{Code: code, Name: "Saw", MachineTypeCode: mt.Code, Capacity: 120, CapacityUnit: types.Pieces, CapacityPeriod: types.Hour, EfficiencyRate: 0.5, AvailableHoursPerDay: &hours, IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO items(code,business_code,warehouse_code,health,created_by,enterprise_id) VALUES($1,'CAP/TEST',0,'ATIVO',$2,$3)`, code, actor, enterprise)
	profile := &machineentity.ItemMachineTime{ItemCode: code, MachineCode: code, ProductionTime: 60, ProductionTimeUnit: types.Minute, ProductionBaseQty: 1, SetupTime: 15, Priority: 1, EfficiencyRate: &eff, TimeBasis: "CYCLE"}
	if _, err = machines.CreateItemMachineTime(ctx, profile); err != nil {
		t.Fatal(err)
	}
	// Updating must persist the period AND base quantity, not only the raw time.
	profile.ProductionTime = 1
	profile.ProductionTimeUnit = types.Hour
	profile.ProductionBaseQty = 120
	profile.TimeBasis = "PROPORTIONAL"
	saved, err := machines.CreateItemMachineTime(ctx, profile)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ProductionTimeUnit != types.Hour || saved.ProductionBaseQty != 120 || saved.TimeBasis != "PROPORTIONAL" || saved.EfficiencyRate == nil || !saved.IsActive {
		t.Fatalf("roundtrip: %+v", saved)
	}
	simulation := machinesvc.CalculateProductionTime(saved, m, 60, 1, hours*60)
	if simulation.TotalMinutes != 52.5 {
		t.Fatalf("simulation %v", simulation.TotalMinutes)
	}
	exec(`INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'Capacity plan',$2,$3)`, code, actor, enterprise)
	var suggestion int64
	if err = pool.QueryRow(ctx, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,start_date,order_type,enterprise_id) VALUES($1,$1,60,'2026-09-14','2026-09-14','FABRICACAO',$2) RETURNING code`, code, enterprise).Scan(&suggestion); err != nil {
		t.Fatal(err)
	}
	repo := mrprepo.NewMRPCalculationRepositorySQLC(q, pool)
	svc := &MRPServiceImpl{MRPRepo: repo}
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err != nil {
		t.Fatal(err)
	}
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err != nil {
		t.Fatal(err)
	}
	var duration float64
	var count int
	if err = pool.QueryRow(ctx, `SELECT production_minutes FROM mrp_machine_allocations WHERE suggestion_code=$1`, suggestion).Scan(&duration); err != nil {
		t.Fatal(err)
	}
	if duration != simulation.TotalMinutes {
		t.Fatalf("MRP %v != simulator %v", duration, simulation.TotalMinutes)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM machine_schedules WHERE enterprise_id=$1`, enterprise).Scan(&count); err != nil || count != 0 {
		t.Fatalf("unreleased suggestion entered execution queue: %d %v", count, err)
	}
	day := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	h, err := capacity.Hours(ctx, pool, mt.ID, &day)
	if err != nil || h != 6 {
		t.Fatalf("capacity %v %v", h, err)
	}
	crp := crp_uc.New(crprepo.New(q, pool)).WithPlanCatalog(q)
	summary, err := crp.CalculateCRP(ctx, request.CalculateCRPDTO{PlanCode: code})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalEntries != 1 || summary.OverloadCount != 0 {
		t.Fatalf("CRP %+v", summary)
	}
	entries, err := crp.ListByPlan(ctx, code)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries %v %v", entries, err)
	}
	if math.Abs(entries[0].RequiredHours-0.875) > 1e-6 || entries[0].AvailableHours != 6 {
		t.Fatalf("load %+v", entries[0])
	}
	// Closed calendar means no capacity; overlaps must not count twice.
	var cal int64
	if err = pool.QueryRow(ctx, `INSERT INTO machine_calendars(enterprise_id,code,description) VALUES($1,$2,'Shift') RETURNING id`, enterprise, code).Scan(&cal); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE machines SET calendar_id=$1 WHERE id=$2`, cal, m.ID)
	h, err = capacity.Hours(ctx, pool, mt.ID, &day)
	if err != nil || h != 0 {
		t.Fatalf("closed calendar %v %v", h, err)
	}
	exec(`INSERT INTO machine_calendar_intervals(calendar_id,weekday,start_time,end_time) VALUES($1,1,'08:00','12:00'),($1,1,'10:00','14:00')`, cal)
	h, err = capacity.Hours(ctx, pool, mt.ID, &day)
	if err != nil || h != 6 {
		t.Fatalf("overlapping shifts %v %v", h, err)
	}
	exec(`INSERT INTO machine_downtimes(enterprise_id,machine_id,starts_at,ends_at,downtime_type,reason) VALUES($1,$2,'2026-09-14 09:00','2026-09-14 10:00','PLANNED','test')`, enterprise, m.ID)
	h, err = capacity.Hours(ctx, pool, mt.ID, &day)
	if err != nil || h != 5 {
		t.Fatalf("downtime %v %v", h, err)
	}
	other := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise + 999999})
	times, err := repo.ListItemMachineTimes(other, []int64{code})
	if err != nil || len(times) != 0 {
		t.Fatalf("tenant leakage %v %v", times, err)
	}
	h, err = capacity.Hours(other, pool, mt.ID, &day)
	if err != nil || h != 0 {
		t.Fatalf("capacity tenant leakage %v %v", h, err)
	}
	// Recalculate two orders on a single shift with one hour blocked.
	var second int64
	if err = pool.QueryRow(ctx, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,start_date,order_type,enterprise_id) VALUES($1,$1,600,'2026-09-14','2026-09-14','FABRICACAO',$2) RETURNING code`, code, enterprise).Scan(&second); err != nil {
		t.Fatal(err)
	}
	// Open the next day so the long order can spill over.
	exec(`INSERT INTO machine_calendar_intervals(calendar_id,weekday,start_time,end_time) VALUES($1,2,'08:00','14:00')`, cal)
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err != nil {
		t.Fatal(err)
	}
	var overlaps int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM mrp_machine_allocation_slots a JOIN mrp_machine_allocation_slots b ON a.suggestion_code<b.suggestion_code AND a.starts_at<b.ends_at AND a.ends_at>b.starts_at WHERE a.suggestion_code=$1 AND b.suggestion_code=$2`, suggestion, second).Scan(&overlaps); err != nil || overlaps != 0 {
		t.Fatalf("overlap %d %v", overlaps, err)
	}
	projected, err := repo.GetSuggestionByCode(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if !projected.CapacityLate || projected.EstimatedEndAt == nil || projected.EstimatedEndAt.Day() != 15 {
		t.Fatalf("expected next-day completion, got %+v", projected)
	}
	if projected.NeedDate.Day() != 14 {
		t.Fatal("original need date changed")
	}
	summary, err = crp.CalculateCRP(ctx, request.CalculateCRPDTO{PlanCode: code})
	if err != nil {
		t.Fatal(err)
	}
	entries, err = crp.ListByPlan(ctx, code)
	if err != nil {
		t.Fatal(err)
	}
	sum := 0.0
	for _, v := range entries {
		sum += v.RequiredHours
		if v.RequiredHours > v.AvailableHours+0.0001 {
			t.Fatalf("finite plan overloaded %+v", v)
		}
	}
	if math.Abs(sum-(52.5+390)/60) > 0.0002 {
		t.Fatalf("CRP lost or duplicated hours %v", sum)
	}

	// A two-operation routing must reserve both steps and carry measured times to OF.
	var routeID, operationID int64
	if err = pool.QueryRow(ctx, `INSERT INTO manufacturing_routes(code,item_code,is_standard,created_by,enterprise_id) VALUES($1,$1,true,$2,$3) RETURNING id`, code, actor, enterprise).Scan(&routeID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO operations(code,name,default_work_center_id,created_by,enterprise_id) VALUES($1,'Cut',$2,$3,$4) RETURNING id`, code, mt.ID, actor, enterprise).Scan(&operationID); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,10,$2),($1,20,$2)`, routeID, operationID)
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT production_minutes FROM mrp_machine_allocations WHERE suggestion_code=$1`, suggestion).Scan(&duration); err != nil || math.Abs(duration-105) > 0.0001 {
		t.Fatalf("route duration %v %v", duration, err)
	}
	var plannedID, productionID int64
	if err = pool.QueryRow(ctx, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,demand_code,need_date,plan_code,mrp_suggestion_code,machine_code,created_by,enterprise_id,status) VALUES($1,$1,60,'PRODUCTION',$2,0,'2026-09-14',$1,$3,$1,$4,$5,'RELEASED') RETURNING id`, code, string(types.DemandIndependent), suggestion, actor, enterprise).Scan(&plannedID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,planned_order_id,created_by,enterprise_id) VALUES($1,$1,60,$2,$3,$4) RETURNING id`, code, plannedID, actor, enterprise).Scan(&productionID); err != nil {
		t.Fatal(err)
	}
	opsUC := &productionuc.OrderOperationsUseCase{Q: q, MachinePlanning: repo}
	if _, err = opsUC.ExplodeRoute(ctx, productionID, routeID); err != nil {
		t.Fatal(err)
	}
	if err = repo.ApplyMachinePlanToProduction(ctx, productionID); err != nil {
		t.Fatal(err)
	}
	var measured float64
	if err = pool.QueryRow(ctx, `SELECT SUM(planned_hours+setup_hours) FROM production_order_operations WHERE production_order_id=$1`, productionID).Scan(&measured); err != nil || math.Abs(measured-1.75) > 0.0001 {
		t.Fatalf("OF lost measured time: %v %v", measured, err)
	}
	if err = pool.QueryRow(ctx, `SELECT SUM(EXTRACT(EPOCH FROM(scheduled_end-scheduled_start))/3600) FROM production_sequences WHERE production_order_id=$1 AND machine_id=$2`, productionID, m.ID).Scan(&measured); err != nil || math.Abs(measured-1.75) > 0.0001 {
		t.Fatalf("OF reservations duplicated/lost: %v %v", measured, err)
	}
	aps := apsrepo.New(q, pool).(interface {
		MachinePlanOperationCount(context.Context, int64) (int, error)
	})
	if n, err := aps.MachinePlanOperationCount(ctx, productionID); err != nil || n != 2 {
		t.Fatalf("APS failed to preserve route: %d %v", n, err)
	}
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err != nil {
		t.Fatal(err)
	}
	if _, err = crp.CalculateCRP(ctx, request.CalculateCRPDTO{PlanCode: code}); err != nil {
		t.Fatal(err)
	}
	entries, err = crp.ListByPlan(ctx, code)
	if err != nil {
		t.Fatal(err)
	}
	measured = 0
	for _, v := range entries {
		measured += v.RequiredHours
	}
	if math.Abs(measured-2*(52.5+390)/60) > 0.0003 {
		t.Fatalf("released CRP load changed: %v", measured)
	}
	// A failed replan must retain the previous complete reservations.
	exec(`DELETE FROM machine_calendar_intervals WHERE calendar_id=$1`, cal)
	if err = svc.processMachineIntegration(ctx, code, []int64{code}); err == nil {
		t.Fatal("closed calendar should fail")
	}
	if err = pool.QueryRow(ctx, `SELECT production_minutes FROM mrp_machine_allocations WHERE suggestion_code=$1`, second).Scan(&duration); err != nil || duration != 780 {
		t.Fatalf("failed plan did not roll back: %v %v", duration, err)
	}
	exec(`UPDATE machines SET is_active=false WHERE id=$1`, m.ID)
	times, err = repo.ListItemMachineTimes(ctx, []int64{code})
	if err != nil || len(times) != 0 {
		t.Fatalf("inactive machine selected %v %v", times, err)
	}
}
