package mrp_calculation_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type profileRepo struct {
	mrprepo.MRPCalculationRepository
	rows []*mrpentity.MRPItemProfile
}

func (r profileRepo) GetProfiles(context.Context, int64, int64) ([]*mrpentity.MRPItemProfile, error) {
	return r.rows, nil
}

type profileAuth struct{ ports.AuthService }

func (profileAuth) CanRunMRPCalculation(context.Context) bool { return true }

type profileStock struct{ balances []*stockentity.StockBalance }

func (s profileStock) ListBalancesByItem(context.Context, int64) ([]*stockentity.StockBalance, error) {
	return s.balances, nil
}

func TestProfileConsultation_FiltersAndBuildsCurrentTotals(t *testing.T) {
	d1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	uc := GetItemProfileUseCase{Repo: profileRepo{rows: []*mrpentity.MRPItemProfile{
		{ItemCode: 1, PlanCode: 2, NeedDate: d1, Demand: 5, StockProjected: 10},
		{ItemCode: 1, PlanCode: 2, NeedDate: d2, Demand: 8, StockProjected: 2},
	}}, Auth: profileAuth{}, Stock: profileStock{balances: []*stockentity.StockBalance{{Quantity: 7}, {Quantity: 3}}}}
	to := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	result, err := uc.Consult(context.Background(), 1, 2, "CURRENT", nil, &to)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Totals["demand"] != 5 || result.Totals["stock_current"] != 10 {
		t.Fatalf("unexpected consultation: %+v", result)
	}
}
