package service

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	orderpriority "github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structrepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type publicMethodStructRepo struct {
	structrepo.ItemStructureRepository
	children []*structentity.ItemStructure
	bom      map[int64][]*structentity.ItemStructure
}

func (r *publicMethodStructRepo) GetAllDirectChildren(context.Context, int64) ([]*structentity.ItemStructure, error) {
	return r.children, nil
}
func (r *publicMethodStructRepo) LoadBOMForRoots(context.Context, []int64) (map[int64][]*structentity.ItemStructure, error) {
	return r.bom, nil
}

type publicMethodMRPRepo struct {
	mrprepo.MRPCalculationRepository
	params *entity.TypedPlanningParams
}

func (r *publicMethodMRPRepo) LoadTypedPlanningParams(context.Context) (*entity.TypedPlanningParams, error) {
	return r.params, nil
}

func TestPublicStructureAndLLCMethods(t *testing.T) {
	mask := "A"
	structure := &publicMethodStructRepo{children: []*structentity.ItemStructure{{ChildCode: 2, Quantity: 2, LossPercentage: 10, IsActive: true}, {ChildCode: 3, Quantity: 1, IsActive: false}, {ChildCode: 4, Quantity: 1, IsActive: true, ParentMask: &mask}}, bom: map[int64][]*structentity.ItemStructure{1: {{ChildCode: 2, IsActive: true}}, 2: {{ChildCode: 5, IsActive: true}}}}
	service := &MRPServiceImpl{StructRepo: structure, MRPRepo: &publicMethodMRPRepo{params: &entity.TypedPlanningParams{FormulaPerdasEstrutura: 1}}}
	inputs, err := service.ExplodeStructure(context.Background(), 1, "", 3, 1)
	if err != nil || len(inputs) != 1 || abs(inputs[0].Quantity-6.6) > 1e-6 {
		t.Fatalf("inputs=%+v err=%v", inputs, err)
	}
	if deep, err := service.ExplodeStructure(context.Background(), 1, "", 1, 21); err != nil || len(deep) != 0 {
		t.Fatalf("depth guard=%v err=%v", deep, err)
	}
	llc, err := service.CalculateItemLLC(context.Background(), 1)
	if err != nil || llc != 0 {
		t.Fatalf("llc=%d err=%v", llc, err)
	}
	if err := service.GenerateLLC(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFindPriorityForQuantity(t *testing.T) {
	priorities := []*orderpriority.OrderPriority{
		{IntervalStart: 0, IntervalEnd: 10, Priority: "LOW"},
		{IntervalStart: 11, IntervalEnd: 100, Priority: "MEDIUM"},
		{IntervalStart: 101, IntervalEnd: 1000, Priority: "HIGH"},
	}
	cases := []struct {
		qty  float64
		want string
	}{
		{5, "LOW"},
		{10, "LOW"},    // inclusive upper bound
		{11, "MEDIUM"}, // inclusive lower bound
		{500, "HIGH"},
		{5000, ""}, // outside every interval
	}
	for _, c := range cases {
		if got := findPriorityForQuantity(priorities, c.qty); got != c.want {
			t.Errorf("findPriorityForQuantity(%v) = %q, want %q", c.qty, got, c.want)
		}
	}
}

func TestRuleMatchesMask(t *testing.T) {
	s := &MRPServiceImpl{}
	cases := []struct {
		name     string
		ruleType string
		value    string
		mask     string
		want     bool
	}{
		{"equal hit", "EQUAL", "0010", "0010", true},
		{"equal miss", "EQUAL", "0010", "0020", false},
		{"different hit", "DIFFERENT", "0010", "0020", true},
		{"different miss", "DIFFERENT", "0010", "0010", false},
		{"range in", "RANGE", "10-20", "15", true},
		{"range edge low", "RANGE", "10-20", "10", true},
		{"range edge high", "RANGE", "10-20", "20", true},
		{"range out", "RANGE", "10-20", "25", false},
		{"range malformed", "RANGE", "10..20", "15", false},
		{"range non-numeric mask", "RANGE", "10-20", "abc", false},
		{"unknown type applies always", "WHATEVER", "x", "y", true},
		{"lowercase type normalized", "equal", "0010", "0010", true},
	}
	for _, c := range cases {
		rule := &entity.ConfiguredItemRule{RuleType: c.ruleType, RuleValue: c.value}
		if got := s.ruleMatchesMask(rule, c.mask); got != c.want {
			t.Errorf("%s: ruleMatchesMask = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestGetLeadTimeDays_FallbackToConfiguredRule(t *testing.T) {
	// RoutingRepo nil → uses the configured-rule fallback (max lead_time value).
	s := &MRPServiceImpl{}
	rulesMap := map[int64][]*entity.ConfiguredItemRule{
		7: {
			{FieldName: "lead_time", RuleValue: "3"},
			{FieldName: "lead_time", RuleValue: "5"}, // max wins
			{FieldName: "other", RuleValue: "99"},    // ignored
			{FieldName: "lead_time", RuleValue: "x"}, // unparsable → ignored
		},
	}
	if got := s.getLeadTimeDays(rulesMap, 7); got != 5 {
		t.Fatalf("lead time = %d, want 5 (max configured)", got)
	}
	if got := s.getLeadTimeDays(rulesMap, 999); got != 0 {
		t.Fatalf("unknown item lead time = %d, want 0", got)
	}
}

func TestExplodeFromBOM_DefaultFormulaAppliesLoss(t *testing.T) {
	// Default formula (1): qty = parentQty * childQtyPer * (1 + loss/100).
	bom := map[int64][]*structentity.ItemStructure{
		100: {
			{ChildCode: 200, Quantity: 2, LossPercentage: 10}, // 5*2*1.1 = 11
			{ChildCode: 300, Quantity: 1, LossPercentage: 0},  // 5*1     = 5
		},
	}
	out := explodeFromBOM(bom, 100, "", 5, 0)
	if len(out) != 2 {
		t.Fatalf("expected 2 children, got %d", len(out))
	}
	byItem := map[int64]float64{}
	for _, in := range out {
		byItem[in.ItemCode] = in.Quantity
	}
	if byItem[200] < 10.999 || byItem[200] > 11.001 {
		t.Fatalf("child 200 qty = %v, want ~11", byItem[200])
	}
	if byItem[300] != 5 {
		t.Fatalf("child 300 qty = %v, want 5", byItem[300])
	}
}

func TestMaxLLC(t *testing.T) {
	if got := maxLLC(map[int64]int{1: 0, 2: 3, 3: 1}); got != 3 {
		t.Fatalf("maxLLC = %d, want 3", got)
	}
	if got := maxLLC(map[int64]int{}); got != 0 {
		t.Fatalf("maxLLC(empty) = %d, want 0", got)
	}
}

func TestMpsPeriodToDate(t *testing.T) {
	month := mpsPeriodToDate("MONTH", 3, 2026)
	if month.Year() != 2026 || month.Month() != time.March || month.Day() != 1 {
		t.Fatalf("MONTH period = %v, want 2026-03-01", month)
	}
	def := mpsPeriodToDate("UNKNOWN", 9, 2026)
	if def.Month() != time.January || def.Day() != 1 {
		t.Fatalf("default period = %v, want 2026-01-01", def)
	}
	// WEEK delegates to mrpWeekToDate → always a Monday.
	wk := mpsPeriodToDate("week", 1, 2026)
	if wk.Weekday() != time.Monday {
		t.Fatalf("WEEK period should be a Monday, got %v (%v)", wk.Weekday(), wk)
	}
}

func TestMrpWeekToDate_IsoWeekMonday(t *testing.T) {
	d := mrpWeekToDate(2026, 1)
	if d.Weekday() != time.Monday {
		t.Fatalf("week start should be Monday, got %v", d.Weekday())
	}
	// Week 10 is 9 weeks (63 days) after week 1.
	d10 := mrpWeekToDate(2026, 10)
	if diff := d10.Sub(d).Hours() / 24; diff != 63 {
		t.Fatalf("weeks 1→10 gap = %v days, want 63", diff)
	}
}
