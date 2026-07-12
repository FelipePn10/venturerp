package entity

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func newValidPlan(t *testing.T, independent string, types []string) *ProductionPlan {
	t.Helper()
	p, err := NewProductionPlan(1, " Plano principal ", independent, true, types, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProductionPlanNormalizesPlanningTypesAndClassCodes(t *testing.T) {
	p := newValidPlan(t, IndependentDemandsAll, []string{"kanban", "MRP", "kanban"})
	classification, codes := "  grupo  ", "3, 1,3,2"
	if err := p.Configure(&classification, &codes, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := p.ClassItemCodes; got == nil || *got != "1,2,3" {
		t.Fatalf("unexpected class codes: %v", got)
	}
	if len(p.PlanningTypes) != 2 || p.PlanningTypes[0] != "KANBAN" || p.PlanningTypes[1] != "MRP" {
		t.Fatalf("unexpected types: %v", p.PlanningTypes)
	}
}

func TestProductionPlanAcceptsHierarchicalClassCodes(t *testing.T) {
	p := newValidPlan(t, IndependentDemandsAll, nil)
	classification, codes := "10", "10.200, 10.100.300"
	if err := p.Configure(&classification, &codes, nil, nil); err != nil {
		t.Fatal(err)
	}
	if p.ClassItemCodes == nil || *p.ClassItemCodes != "10.100.300,10.200" {
		t.Fatalf("unexpected hierarchical class codes: %v", p.ClassItemCodes)
	}
}

func TestProductionPlanRequiresFromDate(t *testing.T) {
	p := newValidPlan(t, IndependentDemandsFromDate, nil)
	if err := p.Configure(nil, nil, nil, nil); !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("expected invalid plan, got %v", err)
	}
	if err := p.Configure(nil, nil, nil, map[string]interface{}{"from_date": "2026-07-10"}); err != nil {
		t.Fatal(err)
	}
}

func TestProductionPlanRejectsAmbiguousOrderAndClassificationFilters(t *testing.T) {
	p := newValidPlan(t, IndependentDemandsNo, nil)
	classification := "grupo"
	order := int64(10)
	if err := p.Configure(&classification, nil, &order, nil); !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("expected invalid plan, got %v", err)
	}
}

func TestProductionPlanRejectsUnsupportedPlanningType(t *testing.T) {
	_, err := NewProductionPlan(1, "Plano", IndependentDemandsAll, false, []string{"DRP"}, uuid.New())
	if !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("expected invalid plan, got %v", err)
	}
}
