package sales_forecast_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type actualsAuth struct{ ports.AuthService }

func (actualsAuth) CanListSalesForecasts(context.Context) bool { return true }

type actualsRepo struct {
	repository.SalesForecastRepository
	source string
	items  []int64
}

func (r *actualsRepo) ListHistoricalDemand(_ context.Context, source string, _, _ time.Time, items []int64) ([]*entity.HistoricalDemand, error) {
	r.source, r.items = source, items
	return []*entity.HistoricalDemand{{ItemCode: 10, PeriodMonth: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Quantity: 42}}, nil
}

func TestListActualDemandReturnsMonthlyActuals(t *testing.T) {
	repo := &actualsRepo{}
	uc := &ListActualDemandUseCase{Repo: repo, Auth: actualsAuth{}}
	got, err := uc.Execute(context.Background(), 2026, 10, "invoicing")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Month != 3 || got[0].Quantity != 42 || repo.source != "INVOICING" || len(repo.items) != 1 || repo.items[0] != 10 {
		t.Fatalf("resultado inesperado: %#v source=%s items=%v", got, repo.source, repo.items)
	}
}

func TestListActualDemandRejectsInvalidFilters(t *testing.T) {
	uc := &ListActualDemandUseCase{Repo: &actualsRepo{}, Auth: actualsAuth{}}
	for _, tc := range []struct {
		year   int
		source string
	}{{0, "ORDERS"}, {2026, "UNKNOWN"}} {
		if _, err := uc.Execute(context.Background(), tc.year, 0, tc.source); err == nil {
			t.Fatalf("aceitou filtro inválido: %+v", tc)
		}
	}
}
