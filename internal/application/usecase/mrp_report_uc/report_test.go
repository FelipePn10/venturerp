package mrp_report_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type reportAuth struct {
	ports.AuthService
	allowed bool
}

func (a reportAuth) CanRunMRPCalculation(context.Context) bool { return a.allowed }

type reportReader struct {
	calls []string
	err   error
}

func (r *reportReader) add(name string) ([]ReportRow, error) {
	r.calls = append(r.calls, name)
	return []ReportRow{{ItemCode: 1}}, r.err
}
func (r *reportReader) Profile(context.Context, Filter) ([]ReportRow, error) { return r.add("profile") }
func (r *reportReader) Availability(context.Context, Filter) ([]ReportRow, error) {
	return r.add("availability")
}
func (r *reportReader) GroupedNeeds(context.Context, Filter) ([]ReportRow, error) {
	return r.add("grouped")
}
func (r *reportReader) Explosion(context.Context, int64, decimal.Decimal, *time.Time, Filter) ([]ReportRow, error) {
	return r.add("explosion")
}
func (r *reportReader) ReorderPoint(context.Context, Filter) ([]ReportRow, error) {
	return r.add("reorder")
}

func TestReportUseCaseAuthorizesAndDelegatesEveryReport(t *testing.T) {
	reader := &reportReader{}
	uc := UseCase{Reader: reader, Auth: reportAuth{allowed: true}}
	ctx := context.Background()
	itemCode := int64(1)
	planCode := int64(1)
	filter := Filter{PlanCode: &planCode, ItemCode: &itemCode, Quantity: decimal.NewFromInt(1)}
	checks := []func() error{
		func() error { _, e := uc.Profile(ctx, filter); return e }, func() error { _, e := uc.Availability(ctx, filter); return e },
		func() error { _, e := uc.GroupedNeeds(ctx, filter); return e }, func() error { _, e := uc.Explosion(ctx, 1, decimal.NewFromInt(1), nil, filter); return e },
		func() error { _, e := uc.ReorderPoint(ctx, filter); return e }}
	for _, check := range checks {
		if err := check(); err != nil {
			t.Fatal(err)
		}
	}
	if len(reader.calls) != 5 {
		t.Fatalf("delegations=%v", reader.calls)
	}
}
func TestReportUseCaseRejectsUnauthorizedAndInvalidExplosion(t *testing.T) {
	ctx := context.Background()
	reader := &reportReader{}
	unauthorized := UseCase{Reader: reader, Auth: reportAuth{allowed: false}}
	if _, err := unauthorized.Profile(ctx, Filter{}); err == nil {
		t.Fatal("expected unauthorized")
	}
	allowed := UseCase{Reader: reader, Auth: reportAuth{allowed: true}}
	if _, err := allowed.Explosion(ctx, 0, decimal.Zero, nil, Filter{}); err == nil {
		t.Fatal("expected validation")
	}
	reader.err = errors.New("reader failure")
	if _, err := allowed.ReorderPoint(ctx, Filter{}); !errors.Is(err, reader.err) {
		t.Fatalf("err=%v", err)
	}
}

func TestReportUseCaseValidatesDetailedFilters(t *testing.T) {
	ctx := context.Background()
	uc := UseCase{Reader: &reportReader{}, Auth: reportAuth{allowed: true}}
	mask := int64(1)
	plan := int64(1)
	from := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cases := []Filter{
		{From: &from, To: &to},
		{ClassificationMaskCode: &mask},
		{ClassificationCode: "10%"},
		{ItemType: "INVALID"},
		{BreakBy: "INVALID"},
		{Layout: "AMBOS"},
		{Periods: []DateRange{{From: from, To: to}}},
	}
	for i, filter := range cases {
		if _, err := uc.Profile(ctx, filter); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
	if _, err := uc.Profile(ctx, Filter{PlanCode: &plan, ClassificationMaskCode: &mask, ClassificationCode: "10%", Layout: "analitico", BreakBy: "item", ItemType: "fabricado"}); err != nil {
		t.Fatalf("valid detailed filter: %v", err)
	}
}

func TestAggregatePeriodsProducesSixOrderedColumns(t *testing.T) {
	jan := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	periods := make([]DateRange, 6)
	for i := range periods {
		periods[i] = DateRange{From: time.Date(2026, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, time.Month(i+2), 0, 0, 0, 0, 0, time.UTC)}
	}
	rows := aggregatePeriods([]ReportRow{{ItemCode: 1, Date: &jan, Required: decimal.NewFromInt(2)}, {ItemCode: 1, Date: &feb, Required: decimal.NewFromInt(3)}}, periods)
	if len(rows) != 1 || len(rows[0].PeriodValues) != 6 {
		t.Fatalf("rows=%+v", rows)
	}
	if !rows[0].PeriodValues[0].Equal(decimal.NewFromInt(2)) || !rows[0].PeriodValues[1].Equal(decimal.NewFromInt(3)) || !rows[0].PeriodValues[5].IsZero() {
		t.Fatalf("periods=%v", rows[0].PeriodValues)
	}
}
