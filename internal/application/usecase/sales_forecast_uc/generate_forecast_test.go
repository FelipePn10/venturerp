package sales_forecast_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	calendarentity "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/google/uuid"
)

type forecastAuth struct{ ports.AuthService }

func (forecastAuth) CanCreateSalesForecast(context.Context) bool { return true }
func (forecastAuth) UserID(context.Context) (uuid.UUID, error)   { return uuid.New(), nil }

type memoryForecastRepo struct {
	forecasts []*entity.SalesForecast
	blocked   map[string]bool
	history   []*entity.HistoricalDemand
	nextID    int64
}

func newMemoryForecastRepo() *memoryForecastRepo {
	return &memoryForecastRepo{blocked: map[string]bool{}, nextID: 1}
}

func (r *memoryForecastRepo) CreateForecast(ctx context.Context, f *entity.SalesForecast) (*entity.SalesForecast, error) {
	_ = ctx
	cp := *f
	cp.ID = r.nextID
	r.nextID++
	r.forecasts = append(r.forecasts, &cp)
	return &cp, nil
}

func (r *memoryForecastRepo) UpdateForecast(ctx context.Context, f *entity.SalesForecast) (*entity.SalesForecast, error) {
	_ = ctx
	for _, current := range r.forecasts {
		if current.ID == f.ID {
			current.Quantity = f.Quantity
			current.UpdatedAt = time.Now()
			cp := *current
			return &cp, nil
		}
	}
	return f, nil
}

func (r *memoryForecastRepo) GetForecastByItem(ctx context.Context, itemCode int64) ([]*entity.SalesForecast, error) {
	_ = ctx
	var out []*entity.SalesForecast
	for _, forecast := range r.forecasts {
		if forecast.ItemCode == itemCode {
			out = append(out, forecast)
		}
	}
	return out, nil
}

func (r *memoryForecastRepo) ListForecasts(context.Context, int) ([]*entity.SalesForecast, error) {
	return r.forecasts, nil
}

func (r *memoryForecastRepo) ListHistoricalDemand(context.Context, string, time.Time, time.Time, []int64) ([]*entity.HistoricalDemand, error) {
	return r.history, nil
}

func (r *memoryForecastRepo) DeleteForecast(context.Context, int64) error { return nil }
func (r *memoryForecastRepo) CreateBlock(context.Context, *entity.SalesForecastBlock) (*entity.SalesForecastBlock, error) {
	return nil, nil
}
func (r *memoryForecastRepo) ListBlocks(context.Context) ([]*entity.SalesForecastBlock, error) {
	return nil, nil
}
func (r *memoryForecastRepo) IsBlocked(ctx context.Context, date time.Time) (bool, error) {
	_ = ctx
	year, week := date.ISOWeek()
	return r.blocked[forecastPeriodKey(year, week)], nil
}
func (r *memoryForecastRepo) DeleteBlock(context.Context, int64) error { return nil }
func (r *memoryForecastRepo) CreateAppropriation(context.Context, *entity.AppropriationTable) (*entity.AppropriationTable, error) {
	return nil, nil
}
func (r *memoryForecastRepo) UpdateAppropriation(context.Context, *entity.AppropriationTable) (*entity.AppropriationTable, error) {
	return nil, nil
}
func (r *memoryForecastRepo) GetDefaultAppropriation(context.Context) (*entity.AppropriationTable, error) {
	return nil, nil
}
func (r *memoryForecastRepo) ListAppropriations(context.Context) ([]*entity.AppropriationTable, error) {
	return nil, nil
}
func (r *memoryForecastRepo) SetDefaultAppropriation(context.Context, int64) error { return nil }

type calendarFake struct {
	workdays map[string][]int
}

func (c calendarFake) CreateDay(context.Context, *calendarentity.IndustrialCalendar) (*calendarentity.IndustrialCalendar, error) {
	return nil, nil
}
func (c calendarFake) GetDay(context.Context, int, int, int) (*calendarentity.IndustrialCalendar, error) {
	return nil, nil
}
func (c calendarFake) GetWorkdaysInMonth(_ context.Context, year, month int) ([]*calendarentity.IndustrialCalendar, error) {
	var out []*calendarentity.IndustrialCalendar
	for _, day := range c.workdays[monthKey(year, month)] {
		out = append(out, &calendarentity.IndustrialCalendar{Year: year, Month: month, Day: day, IsWorkday: true})
	}
	return out, nil
}
func (c calendarFake) IsWorkday(context.Context, int, int, int) (bool, error) { return true, nil }
func (c calendarFake) GetNextWorkday(context.Context, int, int, int) (time.Time, error) {
	return time.Time{}, nil
}
func (c calendarFake) ListMonth(context.Context, int, int) ([]*calendarentity.IndustrialCalendar, error) {
	return nil, nil
}
func (c calendarFake) DeleteDay(context.Context, int, int, int) error { return nil }
func (c calendarFake) SubtractWorkdays(context.Context, time.Time, int) (time.Time, error) {
	return time.Time{}, nil
}

func monthKey(year, month int) string {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}

func TestGenerateSalesForecastCreatesAndSkipsBlockedPeriod(t *testing.T) {
	repo := newMemoryForecastRepo()
	repo.blocked[forecastPeriodKey(2026, 28)] = true
	uc := GenerateSalesForecastUseCase{Repo: repo, Auth: forecastAuth{}}

	got, err := uc.Execute(context.Background(), generateForecastDTO(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GeneratedCount != 2 {
		t.Fatalf("generated count = %d, want 2", got.GeneratedCount)
	}
	if len(got.Created) != 2 {
		t.Fatalf("created = %d, want 2", len(got.Created))
	}
	if len(got.Skipped) != 1 || got.Skipped[0].Week != 28 {
		t.Fatalf("skipped = %#v, want blocked week 28", got.Skipped)
	}
}

func TestGenerateSalesForecastUpdatesExistingWhenRequested(t *testing.T) {
	repo := newMemoryForecastRepo()
	existing, err := entity.NewSalesForecast(1001, nil, 27, 2026, 10, uuid.New())
	if err != nil {
		t.Fatalf("seed forecast: %v", err)
	}
	_, _ = repo.CreateForecast(context.Background(), existing)
	uc := GenerateSalesForecastUseCase{Repo: repo, Auth: forecastAuth{}}

	got, err := uc.Execute(context.Background(), generateForecastDTO(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Updated) != 1 {
		t.Fatalf("updated = %d, want 1", len(got.Updated))
	}
	if got.Updated[0].Week != 27 {
		t.Fatalf("updated week = %d, want 27", got.Updated[0].Week)
	}
}

func TestGenerateSalesForecastSkipsExistingWhenUpdateDisabled(t *testing.T) {
	repo := newMemoryForecastRepo()
	existing, err := entity.NewSalesForecast(1001, nil, 27, 2026, 10, uuid.New())
	if err != nil {
		t.Fatalf("seed forecast: %v", err)
	}
	_, _ = repo.CreateForecast(context.Background(), existing)
	uc := GenerateSalesForecastUseCase{Repo: repo, Auth: forecastAuth{}}

	got, err := uc.Execute(context.Background(), generateForecastDTO(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Updated) != 0 {
		t.Fatalf("updated = %d, want 0", len(got.Updated))
	}
	if len(got.Skipped) == 0 || got.Skipped[0].Reason != "forecast already exists" {
		t.Fatalf("skipped = %#v, want existing forecast skip", got.Skipped)
	}
}

func TestCreateMonthlyForecastDistributesByIndustrialWorkdaysAndPutsResidualOnLastWeek(t *testing.T) {
	repo := newMemoryForecastRepo()
	cal := calendarFake{workdays: map[string][]int{
		monthKey(2026, 2): {2, 3, 4, 5, 6, 9, 10, 11},
	}}
	uc := CreateMonthlySalesForecastUseCase{Repo: repo, Calendar: cal, Auth: forecastAuth{}}

	got, err := uc.Execute(context.Background(), request.CreateMonthlySalesForecastDTO{
		ItemCode:        1001,
		Year:            2026,
		Month:           2,
		Quantity:        10,
		AcceptsFraction: false,
		UpdateExisting:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GeneratedCount != 2 {
		t.Fatalf("generated count = %d, want 2", got.GeneratedCount)
	}
	var week6, week7 float64
	for _, created := range got.Created {
		if created.Week == 6 {
			week6 = created.Quantity
		}
		if created.Week == 7 {
			week7 = created.Quantity
		}
	}
	if week6 != 6 || week7 != 4 {
		t.Fatalf("weekly quantities week6=%v week7=%v, want 6 and 4", week6, week7)
	}
}

func TestGenerateFromERPHistoryAveragesSelectedPeriodAndAppliesProjection(t *testing.T) {
	repo := newMemoryForecastRepo()
	repo.history = []*entity.HistoricalDemand{
		{ItemCode: 1001, PeriodMonth: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Quantity: 90},
		{ItemCode: 1001, PeriodMonth: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Quantity: 110},
	}
	cal := calendarFake{workdays: map[string][]int{
		monthKey(2026, 3): {2, 3, 4, 5, 6},
	}}
	uc := GenerateSalesForecastUseCase{Repo: repo, Calendar: cal, Auth: forecastAuth{}}

	got, err := uc.Execute(context.Background(), request.GenerateSalesForecastDTO{
		HistorySource:   "ORDERS",
		HistoryFrom:     "2026-01-01",
		HistoryTo:       "2026-02-28",
		StartWeek:       10,
		StartYear:       2026,
		TargetEndWeek:   10,
		TargetEndYear:   2026,
		ItemCodes:       []int64{1001},
		ProjectionPct:   10,
		AcceptsFraction: true,
		UpdateExisting:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GeneratedCount != 1 {
		t.Fatalf("generated count = %d, want 1", got.GeneratedCount)
	}
	if got.Created[0].Quantity != 110 {
		t.Fatalf("quantity = %v, want projected average 110", got.Created[0].Quantity)
	}
}

func generateForecastDTO(updateExisting bool) request.GenerateSalesForecastDTO {
	return request.GenerateSalesForecastDTO{
		ItemCode:       1001,
		StartWeek:      27,
		StartYear:      2026,
		Periods:        3,
		Model:          "MOVING_AVERAGE",
		MAWindow:       3,
		UpdateExisting: updateExisting,
		History: []request.DataPoint{
			{Period: "2026-04", Quantity: 90},
			{Period: "2026-05", Quantity: 100},
			{Period: "2026-06", Quantity: 110},
			{Period: "2026-07", Quantity: 120},
		},
	}
}
