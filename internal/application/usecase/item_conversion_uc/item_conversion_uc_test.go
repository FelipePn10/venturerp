package item_conversion_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
)

// fakeRepo is an in-memory ItemConversionRepository for tests.
type fakeRepo struct {
	rows              []*entity.ItemUnitConversion
	acceptsFractional bool
}

func (r *fakeRepo) Create(ctx context.Context, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error) {
	r.rows = append(r.rows, c)
	return c, nil
}
func (r *fakeRepo) ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	var out []*entity.ItemUnitConversion
	for _, c := range r.rows {
		if c.ItemCode == itemCode {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *fakeRepo) Get(ctx context.Context, itemCode int64, from, to string) (*entity.ItemUnitConversion, error) {
	for _, c := range r.rows {
		if c.ItemCode == itemCode && c.FromUOM == from && c.ToUOM == to {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeRepo) Delete(ctx context.Context, id int64) error { return nil }
func (r *fakeRepo) GetConfigured(ctx context.Context, itemCode int64, mask, from, to string) (*entity.ItemUnitConversion, error) {
	for _, c := range r.rows {
		if c.ItemCode == itemCode && c.Mask == mask && c.FromUOM == from && c.ToUOM == to {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeRepo) AcceptsFractional(context.Context, int64) (bool, error) {
	return r.acceptsFractional, nil
}

func newUC(t *testing.T) *ItemConversionUseCase {
	t.Helper()
	repo := &fakeRepo{rows: []*entity.ItemUnitConversion{
		{ItemCode: 1, FromUOM: "CX", ToUOM: "UN", Factor: 12}, // 1 CX = 12 UN
	}}
	return NewItemConversionUseCase(repo)
}

func TestFactor_Direct(t *testing.T) {
	uc := newUC(t)
	f, found, err := uc.Factor(context.Background(), 1, "CX", "UN")
	if err != nil || !found {
		t.Fatalf("expected found, err=%v found=%v", err, found)
	}
	if f != 12 {
		t.Errorf("factor = %v, want 12", f)
	}
}

func TestFactor_Inverse(t *testing.T) {
	uc := newUC(t)
	f, found, err := uc.Factor(context.Background(), 1, "UN", "CX")
	if err != nil || !found {
		t.Fatalf("expected found via inverse, err=%v found=%v", err, found)
	}
	if f != 1.0/12.0 {
		t.Errorf("inverse factor = %v, want %v", f, 1.0/12.0)
	}
}

func TestFactor_SameUnit(t *testing.T) {
	uc := newUC(t)
	f, found, _ := uc.Factor(context.Background(), 1, "UN", "UN")
	if !found || f != 1 {
		t.Errorf("same unit should be factor 1 found, got f=%v found=%v", f, found)
	}
	// case-insensitive + trim
	f, found, _ = uc.Factor(context.Background(), 1, " cx ", "un")
	if !found || f != 12 {
		t.Errorf("normalised lookup failed: f=%v found=%v", f, found)
	}
}

func TestFactor_NotFound(t *testing.T) {
	uc := newUC(t)
	_, found, err := uc.Factor(context.Background(), 1, "KG", "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found for unregistered conversion")
	}
}

func TestConvertQuantityAndPrice(t *testing.T) {
	uc := newUC(t)
	// 3 CX → UN: 3 * 12 = 36
	q, found, _ := uc.ConvertQuantity(context.Background(), 1, 3, "CX", "UN")
	if !found || q != 36 {
		t.Errorf("ConvertQuantity = %v (found=%v), want 36", q, found)
	}
	// price per CX 120 → per UN: 120 / 12 = 10
	p, found, _ := uc.ConvertUnitPrice(context.Background(), 1, 120, "CX", "UN")
	if !found || p != 10 {
		t.Errorf("ConvertUnitPrice = %v (found=%v), want 10", p, found)
	}
}

func TestConvertQuantityAppliesRoundingAndTolerancePolicy(t *testing.T) {
	repo := &fakeRepo{rows: []*entity.ItemUnitConversion{{ItemCode: 1, FromUOM: "UN", ToUOM: "CX", Factor: 0.167, RoundingPercent: 50, ToleranceType: "VALUE"}}}
	got, found, err := NewItemConversionUseCase(repo).ConvertQuantityConfigured(context.Background(), 1, "", 6, "UN", "CX")
	if err != nil || !found || got != 1 {
		t.Fatalf("converted=%v found=%v err=%v", got, found, err)
	}
	repo.rows[0].RoundingPercent = 0
	if _, _, err = NewItemConversionUseCase(repo).ConvertQuantityConfigured(context.Background(), 1, "", 6, "UN", "CX"); err == nil {
		t.Fatal("fractional quantity outside policy must be rejected")
	}
}

func TestFactorConfiguredUsesMaskBeforeGenericConversion(t *testing.T) {
	repo := &fakeRepo{rows: []*entity.ItemUnitConversion{
		{ItemCode: 1, Mask: "", FromUOM: "CX", ToUOM: "UN", Factor: 10},
		{ItemCode: 1, Mask: "BLUE", FromUOM: "CX", ToUOM: "UN", Factor: 12},
	}}
	factor, found, err := NewItemConversionUseCase(repo).FactorConfigured(context.Background(), 1, "BLUE", "CX", "UN")
	if err != nil || !found || factor != 12 {
		t.Fatalf("factor=%v found=%v err=%v", factor, found, err)
	}
}
