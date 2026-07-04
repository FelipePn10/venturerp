package cost_uc

import (
	"context"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	scentity "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	"github.com/google/uuid"
)

func TestSelectPrimaryCostSubstitutes(t *testing.T) {
	children := []domainrepo.BOMChild{
		{ChildCode: 20, Quantity: 2, SubstituteGroup: 1, SubstitutePriority: 2},
		{ChildCode: 10, Quantity: 3, SubstituteGroup: 1, SubstitutePriority: 1},
		{ChildCode: 30, Quantity: 5},
	}

	selected := selectPrimaryCostSubstitutes(children)

	got := map[int64]float64{}
	for _, child := range selected {
		got[child.ChildCode] = child.Quantity
	}
	if _, ok := got[20]; ok {
		t.Error("substituto secundário não deve entrar no custo padrão")
	}
	if got[10] != 3 {
		t.Errorf("substituto primário = %v, want 3", got[10])
	}
	if got[30] != 5 {
		t.Errorf("componente standalone = %v, want 5", got[30])
	}
	if len(selected) != 2 {
		t.Errorf("selected = %d, want 2", len(selected))
	}
}

func TestRollUp_UsesMaskResolvedBOM(t *testing.T) {
	repo := &fakeCostRepo{
		childrenByMask: map[string][]domainrepo.BOMChild{
			"1|":  {{ChildCode: 20, Quantity: 99}},
			"1|A": {{ChildCode: 10, Quantity: 2}},
		},
		purchaseCosts: map[int64]float64{
			10: 7,
			20: 1000,
		},
	}
	uc := New(repo)

	res, err := uc.RollUp(context.Background(), request.CostRollupDTO{
		ItemCode:     1,
		Mask:         "A",
		LotSize:      1,
		CalculatedBy: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) == 0 || repo.calls[0] != "1|A" {
		t.Fatalf("first GetDirectChildren call = %v, want 1|A", repo.calls)
	}
	if res.MaterialCost != 14 {
		t.Fatalf("material cost = %v, want 14 from masked child only", res.MaterialCost)
	}
}

type fakeCostRepo struct {
	domainrepo.StandardCostRepository
	childrenByMask map[string][]domainrepo.BOMChild
	purchaseCosts  map[int64]float64
	calls          []string
}

func (f *fakeCostRepo) GetDirectChildren(_ context.Context, parentCode int64, mask string) ([]domainrepo.BOMChild, error) {
	f.calls = append(f.calls, costKey(parentCode, mask))
	return f.childrenByMask[costKey(parentCode, mask)], nil
}

func (f *fakeCostRepo) GetItemPurchaseCost(_ context.Context, itemCode int64) (*scentity.ItemPurchaseCost, error) {
	return &scentity.ItemPurchaseCost{ItemCode: itemCode, UnitCost: f.purchaseCosts[itemCode], Currency: "BRL"}, nil
}

func (f *fakeCostRepo) GetRouteHoursByItem(context.Context, int64, string) (float64, error) {
	return 0, nil
}

func (f *fakeCostRepo) ListWorkCenterCosts(context.Context) ([]*scentity.WorkCenterCost, error) {
	return nil, nil
}

func (f *fakeCostRepo) UpsertItemStandardCost(_ context.Context, cost *scentity.ItemStandardCost) (*scentity.ItemStandardCost, error) {
	cost.TotalCost = cost.MaterialCost + cost.LaborCost + cost.OverheadCost
	return cost, nil
}

func (f *fakeCostRepo) InsertRollupLog(context.Context, *scentity.CostRollupLogEntry) error {
	return nil
}

func costKey(parentCode int64, mask string) string {
	return fmt.Sprintf("%d|%s", parentCode, mask)
}
