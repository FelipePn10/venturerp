package item_conversion_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
)

// fakeRepo é um ManagedItemConversionRepository em memória.
type fakeRepo struct {
	rows              []*entity.ItemUnitConversion
	padrao            []service.Aresta
	acceptsFractional bool
	missing           []string
	events            []*entity.Event
	impacto           repository.Impacto
	unidades          *repository.UnidadesDoItem
	nextID            int64
	inativos          map[int64]bool
}

func (r *fakeRepo) ativo(c *entity.ItemUnitConversion) bool { return !r.inativos[c.ID] }

func (r *fakeRepo) id() int64 { r.nextID++; return r.nextID + 1000 }

func (r *fakeRepo) Create(ctx context.Context, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error) {
	return r.CreateWithEvent(ctx, c, nil)
}
func (r *fakeRepo) CreateWithEvent(_ context.Context, c *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error) {
	cp := *c
	cp.ID = r.id()
	cp.IsActive = true
	r.rows = append(r.rows, &cp)
	if ev != nil {
		r.events = append(r.events, ev)
	}
	return &cp, nil
}
func (r *fakeRepo) ListByItem(_ context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	var out []*entity.ItemUnitConversion
	for _, c := range r.rows {
		if c.ItemCode == itemCode && r.ativo(c) && c.ValidTo == nil {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *fakeRepo) ListVersions(_ context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	var out []*entity.ItemUnitConversion
	for _, c := range r.rows {
		if c.ItemCode == itemCode && r.ativo(c) {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *fakeRepo) Get(_ context.Context, itemCode int64, from, to string) (*entity.ItemUnitConversion, error) {
	for _, c := range r.rows {
		if c.ItemCode == itemCode && c.FromUOM == from && c.ToUOM == to && r.ativo(c) {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeRepo) GetByID(_ context.Context, id int64) (*entity.ItemUnitConversion, error) {
	for _, c := range r.rows {
		if c.ID == id && r.ativo(c) {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeRepo) Update(_ context.Context, c *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error) {
	for i, x := range r.rows {
		if x.ID == c.ID {
			cp := *c
			r.rows[i] = &cp
			r.events = append(r.events, ev)
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeRepo) NewVersion(_ context.Context, atual *entity.ItemUnitConversion, closeAt time.Time, nova *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error) {
	for _, x := range r.rows {
		if x.ID == atual.ID {
			t := closeAt
			x.ValidTo = &t
		}
	}
	cp := *nova
	cp.ID = r.id()
	r.rows = append(r.rows, &cp)
	r.events = append(r.events, ev)
	return &cp, nil
}
func (r *fakeRepo) Deactivate(_ context.Context, id int64, ev *entity.Event) error {
	for _, x := range r.rows {
		if x.ID == id {
			if r.inativos == nil {
				r.inativos = map[int64]bool{}
			}
			r.inativos[id] = true
			r.events = append(r.events, ev)
			return nil
		}
	}
	return errors.New("not found")
}
func (r *fakeRepo) Delete(ctx context.Context, id int64) error { return r.Deactivate(ctx, id, nil) }
func (r *fakeRepo) ListLotConversions(context.Context, int64) ([]*entity.LotConversion, error) {
	return nil, nil
}
func (r *fakeRepo) GetLotConversion(context.Context, int64) (*entity.LotConversion, error) {
	return nil, errors.New("not found")
}
func (r *fakeRepo) UpsertLotConversion(_ context.Context, l *entity.LotConversion, _ *entity.Event) (*entity.LotConversion, error) {
	return l, nil
}
func (r *fakeRepo) DeactivateLotConversion(context.Context, int64, *entity.Event) error { return nil }
func (r *fakeRepo) LotExists(context.Context, int64, string) (bool, error)              { return true, nil }
func (r *fakeRepo) Events(context.Context, int64, int) ([]*entity.Event, error)         { return r.events, nil }
func (r *fakeRepo) Graph(_ context.Context, itemCode int64, mask, _ string, at time.Time) ([]service.Aresta, error) {
	var out []service.Aresta
	for _, c := range r.rows {
		if c.ItemCode != itemCode || !r.ativo(c) || (c.Mask != "" && c.Mask != mask) {
			continue
		}
		if !at.IsZero() && (c.ValidFrom.After(at) || (c.ValidTo != nil && c.ValidTo.Before(at))) {
			continue
		}
		if at.IsZero() && c.ValidTo != nil {
			continue
		}
		de, para := c.FromQty, c.ToQty
		if !de.IsPositive() {
			de, para = decimal.NewFromInt(1), c.Factor
		}
		o := service.OrigemItem
		if c.Mask != "" {
			o = service.OrigemItemMascara
		}
		out = append(out, service.Aresta{ID: c.ID, De: c.FromUOM, Para: c.ToUOM, QtdDe: de, QtdPara: para, Origem: o,
			ArredondamentoPct: c.RoundingPercent, Tolerancia: c.ToleranceValue, TipoTolerancia: c.ToleranceType})
	}
	return append(out, r.padrao...), nil
}
func (r *fakeRepo) AcceptsFractional(context.Context, int64) (bool, error) {
	return r.acceptsFractional, nil
}
func (r *fakeRepo) UnitsOfItem(_ context.Context, itemCode int64) (*repository.UnidadesDoItem, error) {
	if r.unidades != nil {
		return r.unidades, nil
	}
	return &repository.UnidadesDoItem{ItemCode: itemCode, Estoque: "UN"}, nil
}
func (r *fakeRepo) Impact(context.Context, *entity.ItemUnitConversion) (*repository.Impacto, error) {
	imp := r.impacto
	return &imp, nil
}
func (r *fakeRepo) MissingUnits(context.Context, int64, ...string) ([]string, error) {
	return r.missing, nil
}

func newUC(t *testing.T) *ItemConversionUseCase {
	t.Helper()
	repo := &fakeRepo{rows: []*entity.ItemUnitConversion{
		{ItemCode: 1, FromUOM: "CX", ToUOM: "UN", Factor: decimal.NewFromFloat(12)}, // 1 CX = 12 UN
	}}
	return NewItemConversionUseCase(repo, nil)
}

func TestFactor_Direct(t *testing.T) {
	uc := newUC(t)
	f, found, err := uc.Factor(context.Background(), 1, "CX", "UN")
	if err != nil || !found {
		t.Fatalf("expected found, err=%v found=%v", err, found)
	}
	if !f.Equal(decimal.NewFromFloat(12)) {
		t.Errorf("factor = %v, want 12", f)
	}
}

func TestFactor_Inverse(t *testing.T) {
	uc := newUC(t)
	f, found, err := uc.Factor(context.Background(), 1, "UN", "CX")
	if err != nil || !found {
		t.Fatalf("expected found via inverse, err=%v found=%v", err, found)
	}
	// O fator tem a escala da coluna (dez casas); a quantidade usa a fração exata.
	if want := decimal.NewFromInt(1).DivRound(decimal.NewFromInt(12), 10); !f.Equal(want) {
		t.Errorf("inverse factor = %v, want %v", f, want)
	}
}

func TestFactor_SameUnit(t *testing.T) {
	uc := newUC(t)
	f, found, _ := uc.Factor(context.Background(), 1, "UN", "UN")
	if !found || !f.Equal(decimal.NewFromFloat(1)) {
		t.Errorf("same unit should be factor 1 found, got f=%v found=%v", f, found)
	}
	// case-insensitive + trim
	f, found, _ = uc.Factor(context.Background(), 1, " cx ", "un")
	if !found || !f.Equal(decimal.NewFromFloat(12)) {
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
	q, found, _ := uc.ConvertQuantity(context.Background(), 1, decimal.NewFromFloat(3), "CX", "UN")
	if !found || !q.Equal(decimal.NewFromFloat(36)) {
		t.Errorf("ConvertQuantity = %v (found=%v), want 36", q, found)
	}
	// price per CX 120 → per UN: 120 / 12 = 10
	p, found, _ := uc.ConvertUnitPrice(context.Background(), 1, decimal.NewFromFloat(120), "CX", "UN")
	if !found || !p.Equal(decimal.NewFromFloat(10)) {
		t.Errorf("ConvertUnitPrice = %v (found=%v), want 10", p, found)
	}
}

func TestConvertQuantityAppliesRoundingAndTolerancePolicy(t *testing.T) {
	repo := &fakeRepo{rows: []*entity.ItemUnitConversion{{ItemCode: 1, FromUOM: "UN", ToUOM: "CX", Factor: decimal.NewFromFloat(0.167), RoundingPercent: decimal.NewFromFloat(50), ToleranceType: "VALUE"}}}
	got, found, err := NewItemConversionUseCase(repo, nil).ConvertQuantityConfigured(context.Background(), 1, "", decimal.NewFromFloat(6), "UN", "CX")
	if err != nil || !found || !got.Equal(decimal.NewFromFloat(1)) {
		t.Fatalf("converted=%v found=%v err=%v", got, found, err)
	}
	repo.rows[0].RoundingPercent = decimal.NewFromFloat(0)
	if _, _, err = NewItemConversionUseCase(repo, nil).ConvertQuantityConfigured(context.Background(), 1, "", decimal.NewFromFloat(6), "UN", "CX"); err == nil {
		t.Fatal("fractional quantity outside policy must be rejected")
	}
}

func TestFactorConfiguredUsesMaskBeforeGenericConversion(t *testing.T) {
	repo := &fakeRepo{rows: []*entity.ItemUnitConversion{
		{ItemCode: 1, Mask: "", FromUOM: "CX", ToUOM: "UN", Factor: decimal.NewFromFloat(10)},
		{ItemCode: 1, Mask: "BLUE", FromUOM: "CX", ToUOM: "UN", Factor: decimal.NewFromFloat(12)},
	}}
	factor, found, err := NewItemConversionUseCase(repo, nil).FactorConfigured(context.Background(), 1, "BLUE", "CX", "UN")
	if err != nil || !found || !factor.Equal(decimal.NewFromFloat(12)) {
		t.Fatalf("factor=%v found=%v err=%v", factor, found, err)
	}
}
