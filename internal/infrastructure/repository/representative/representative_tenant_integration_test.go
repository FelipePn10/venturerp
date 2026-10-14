//go:build integration

package representative_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	reprepo "github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
	goalentity "github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity"
	goalrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/representative"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_goal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestRepresentativesAndGoalsAreIsolatedByTenant: o representante era um
// cadastro compartilhado e os filhos (telefone, vínculo, região...) eram
// gravados sem conferir a empresa; metas de venda não tinham empresa e o
// relatório somava pedidos de todas.
func TestRepresentativesAndGoalsAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	var repCode int64
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		codes := []int64{a.Code, b.Code}
		_, _ = pool.Exec(base, `DELETE FROM sales_goal_items WHERE goal_code IN (SELECT code FROM sales_goals WHERE enterprise_code = ANY($1))`, codes)
		_, _ = pool.Exec(base, `DELETE FROM sales_goals WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM sales_goal_periods WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM sales_orders WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM representatives WHERE enterprise_code = ANY($1)`, codes)
	})
	reps := representative.New(pool)

	rep, err := reps.Create(a.Ctx, &entity.Representative{Name: "REP DA A", DocumentNumber: "12345678909", IsActive: true, RegisterDate: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	repCode = rep.Code
	if _, err := reps.Get(b.Ctx, repCode); err == nil {
		t.Fatal("empresa B leu o representante da A")
	}
	if l, err := reps.List(b.Ctx, reprepo.RepresentativeFilter{ActiveStatus: "ALL"}); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d representante(s) da A (%v)", len(l), err)
	}
	rep.Name = "ALTERADO POR B"
	if _, err := reps.Update(b.Ctx, rep); err == nil {
		t.Fatal("empresa B alterou o representante da A")
	}
	if err := reps.Block(b.Ctx, repCode, "x"); err == nil {
		t.Fatal("empresa B bloqueou o representante da A")
	}
	if _, err := reps.AddEnterprise(b.Ctx, &entity.RepresentativeEnterprise{RepresentativeCode: repCode, EnterpriseCode: b.Code, IsActive: true}); err == nil {
		t.Fatal("empresa B se vinculou ao representante da A")
	}
	if _, err := reps.AddPhone(b.Ctx, &entity.RepresentativePhone{RepresentativeCode: repCode, Phone: "41999990000", PhoneType: "CELULAR", Ranking: 1}); err == nil {
		t.Fatal("empresa B incluiu telefone no representante da A")
	}
	if _, err := reps.AddRegion(b.Ctx, &entity.RepresentativeRegion{RepresentativeCode: repCode, RegionCode: 1, IsActive: true}); err == nil {
		t.Fatal("empresa B incluiu região no representante da A")
	}
	reg, err := reps.AddRegion(a.Ctx, &entity.RepresentativeRegion{RepresentativeCode: repCode, EnterpriseCode: &b.Code, RegionCode: 1, IsActive: true})
	if err != nil {
		t.Fatalf("empresa A não incluiu região no próprio representante: %v", err)
	}
	if reg.EnterpriseCode == nil || *reg.EnterpriseCode != a.Code {
		t.Fatalf("região do representante da A gravada na empresa %v", reg.EnterpriseCode)
	}
	if _, err := reps.AddPhone(a.Ctx, &entity.RepresentativePhone{RepresentativeCode: repCode, Phone: "41999990000", PhoneType: "CELULAR", Ranking: 1}); err != nil {
		t.Fatalf("empresa A não incluiu telefone no próprio representante: %v", err)
	}

	// Metas
	goals := sales_goal.New(pool)
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	per, err := goals.CreatePeriod(a.Ctx, &goalentity.Period{Description: "JAN A", PeriodType: "MONTH", StartDate: inicio, EndDate: inicio.AddDate(0, 1, -1), IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.GetPeriod(b.Ctx, per.Code); err == nil {
		t.Fatal("empresa B leu o período de metas da A")
	}
	if l, err := goals.ListPeriods(b.Ctx, goalrepo.PeriodFilter{}); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d período(s) da A (%v)", len(l), err)
	}
	if _, err := goals.CreateGoal(b.Ctx, &goalentity.Goal{RepresentativeCode: repCode, PeriodCode: per.Code, AnalysisBase: "SALES", IsActive: true}); err == nil {
		t.Fatal("empresa B criou meta para o representante da A")
	}
	meta, err := goals.CreateGoal(a.Ctx, &goalentity.Goal{RepresentativeCode: repCode, PeriodCode: per.Code, AnalysisBase: "SALES", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.GetGoal(b.Ctx, meta.Code); err == nil {
		t.Fatal("empresa B leu a meta da A")
	}
	if l, err := goals.ListGoals(b.Ctx, goalrepo.GoalFilter{}); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d meta(s) da A (%v)", len(l), err)
	}
	if _, err := goals.UpdateGoal(b.Ctx, meta); err == nil {
		t.Fatal("empresa B alterou a meta da A")
	}
	grupo := int64(7)
	if _, err := goals.AddGoalItem(b.Ctx, &goalentity.GoalItem{GoalCode: meta.Code, TargetType: "GROUP", ItemGroupCode: &grupo, TargetValue: 1}); err == nil {
		t.Fatal("empresa B incluiu item na meta da A")
	}
	if _, err := goals.AddGoalItem(a.Ctx, &goalentity.GoalItem{GoalCode: meta.Code, TargetType: "GROUP", ItemGroupCode: &grupo, TargetValue: 1000, IsActive: true}); err != nil {
		t.Fatalf("empresa A não incluiu item na própria meta: %v", err)
	}

	// Pedido da B com o código do representante da A não conta no realizado da A.
	// É dado legado cruzado (o banco recusa gravar isso desde a 000403).
	testutil.DadoLegadoCruzado(t, pool, "sales_orders", `INSERT INTO sales_orders (order_number, enterprise_code, created_by, representative_code, total_net, emission_date)
		VALUES ($1, $2, $3, $4, 500, $5::date)`, repCode, b.Code, testutil.Actor(t, pool), repCode, inicio.AddDate(0, 0, 5))
	rel, err := goals.Report(a.Ctx, goalrepo.ReportFilter{PeriodCode: &per.Code})
	if err != nil {
		t.Fatal(err)
	}
	if len(rel) != 1 || rel[0].RealizedValue != 0 {
		t.Fatalf("o realizado da A somou pedido da B: %+v", rel)
	}
	if rel2, err := goals.Report(b.Ctx, goalrepo.ReportFilter{}); err != nil || len(rel2) != 0 {
		t.Fatalf("empresa B viu %d linha(s) do relatório de metas da A (%v)", len(rel2), err)
	}
}
