package mrp_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	"github.com/google/uuid"
)

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeMRPRepo struct {
	mrprepo.MRPCalculationRepository
	suggestion *mrpentity.PlannedOrderSuggestion
	errGet     error
}

func (r *fakeMRPRepo) GetSuggestionByCode(_ context.Context, _ int64) (*mrpentity.PlannedOrderSuggestion, error) {
	return r.suggestion, r.errGet
}
func (r *fakeMRPRepo) ListSuggestionsByPlan(_ context.Context, _ int64) ([]*mrpentity.PlannedOrderSuggestion, error) {
	if r.suggestion != nil {
		return []*mrpentity.PlannedOrderSuggestion{r.suggestion}, nil
	}
	return nil, nil
}

type fakePlannedRepo struct {
	plannedrepo.PlannedOrderRepository
	created   *plannedentity.PlannedOrder
	nextNum   int64
	errCreate error
	existing  *plannedentity.PlannedOrder
}

func (r *fakePlannedRepo) GetByMRPSuggestionCode(_ context.Context, _ int64) (*plannedentity.PlannedOrder, error) {
	if r.existing != nil {
		return r.existing, nil
	}
	return nil, errors.New("not found")
}

func (r *fakePlannedRepo) GetNextOrderNumber(_ context.Context) (int64, error) {
	return r.nextNum, nil
}
func (r *fakePlannedRepo) Create(_ context.Context, o *plannedentity.PlannedOrder) (*plannedentity.PlannedOrder, error) {
	if r.errCreate != nil {
		return nil, r.errCreate
	}
	o.Code = 999
	r.created = o
	return o, nil
}

type fakeAuth struct {
	ports.AuthService
	canCreate bool
}

func (a *fakeAuth) CanCreatePlannedOrder(_ context.Context) bool { return a.canCreate }
func (a *fakeAuth) UserID(_ context.Context) (uuid.UUID, error)  { return uuid.Nil, nil }

func baseSuggestion() *mrpentity.PlannedOrderSuggestion {
	nd := time.Now().Add(30 * 24 * time.Hour)
	return &mrpentity.PlannedOrderSuggestion{
		Code:       42,
		PlanCode:   100,
		ItemCode:   2001,
		Quantity:   150,
		OrderType:  "COMPRA", // valores internos do motor MRP (PT); mapeados p/ enum "PURCHASE"
		DemandType: "INDEPENDENTE",
		NeedDate:   nd,
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestFirmarSugestao_CreatesPlannedOrder(t *testing.T) {
	sugg := baseSuggestion()
	reservedNumber := int64(6500)
	warehouseCode := int64(77)
	sugg.OrderNumber = &reservedNumber
	sugg.WarehouseCode = &warehouseCode
	sourceEnterpriseCode := int64(99)
	sugg.InterFactory = true
	sugg.SourceEnterpriseCode = &sourceEnterpriseCode
	sugg.AutoRelease = true
	mrpR := &fakeMRPRepo{suggestion: sugg}
	planR := &fakePlannedRepo{nextNum: 7001}

	uc := &FirmarSugestaoMRPUseCase{
		MRPRepo:     mrpR,
		PlannedRepo: planR,
		Auth:        &fakeAuth{canCreate: true},
	}

	result, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SuggestionCode != 42 {
		t.Errorf("SuggestionCode = %d, want 42", result.SuggestionCode)
	}
	if result.OrderNumber != reservedNumber {
		t.Errorf("OrderNumber = %d, want %d", result.OrderNumber, reservedNumber)
	}
	if result.ItemCode != 2001 {
		t.Errorf("ItemCode = %d, want 2001", result.ItemCode)
	}
	if result.Quantity != 150 {
		t.Errorf("Quantity = %v, want 150", result.Quantity)
	}
	if result.OrderType != "PURCHASE" {
		t.Errorf("OrderType = %q, want PURCHASE", result.OrderType)
	}
	if !result.IsFirm {
		t.Error("created order must be firm")
	}
	if planR.created == nil {
		t.Fatal("no planned order was created")
	}
	pc := planR.created.PlanCode
	if pc == nil || *pc != 100 {
		t.Errorf("PlanCode = %v, want 100", pc)
	}
	if planR.created.WarehouseCode == nil || *planR.created.WarehouseCode != warehouseCode {
		t.Fatalf("WarehouseCode = %v, want %d", planR.created.WarehouseCode, warehouseCode)
	}
	if !planR.created.InterFactory || planR.created.SourceEnterpriseCode == nil || *planR.created.SourceEnterpriseCode != sourceEnterpriseCode || !planR.created.AutoRelease {
		t.Fatalf("inter-factory metadata not propagated: %#v", planR.created)
	}
}

type fakeFirmer struct {
	gotCode int64
	called  bool
}

func (f *fakeFirmer) Execute(_ context.Context, dto request.FirmOrderDTO) (*response.PlannedOrderResponse, error) {
	f.called = true
	f.gotCode = dto.OrderCode
	return &response.PlannedOrderResponse{Code: dto.OrderCode, IsFirm: true}, nil
}
func (f *fakeFirmer) ExecuteTransition(_ context.Context, dto request.TransitionPlannedOrderDTO) ([]*response.PlannedOrderResponse, error) {
	f.called = true
	f.gotCode = dto.OrderCodes[0]
	return []*response.PlannedOrderResponse{{Code: dto.OrderCodes[0], Status: "RELEASED", IsFirm: false}}, nil
}

// With a Firmer wired, accepting a suggestion creates the order NOT firm and then
// firms it (generating the OF/requisition) — a single-step conversion.
func TestFirmarSugestao_ComposesFirmStep(t *testing.T) {
	planR := &fakePlannedRepo{nextNum: 7001}
	firmer := &fakeFirmer{}
	uc := &FirmarSugestaoMRPUseCase{
		MRPRepo:     &fakeMRPRepo{suggestion: baseSuggestion()},
		PlannedRepo: planR,
		Auth:        &fakeAuth{canCreate: true},
		Firmer:      firmer,
	}
	result, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Created NOT firm (so the firm step's first-firming guard fires)...
	if planR.created.IsFirm {
		t.Error("planned order should be created not-firm when a Firmer is wired")
	}
	// ...then firmed via the Firmer on the created code.
	if !firmer.called || firmer.gotCode != 999 {
		t.Errorf("Firmer called=%v code=%d, want true / 999", firmer.called, firmer.gotCode)
	}
	if !result.IsFirm {
		t.Error("result must report the order as firm after the firm step")
	}
}

func TestFirmarSugestao_IsIdempotentBySuggestion(t *testing.T) {
	existing := &plannedentity.PlannedOrder{Code: 77, OrderNumber: 88, ItemCode: 2001, IsFirm: true}
	planR := &fakePlannedRepo{existing: existing}
	uc := &FirmarSugestaoMRPUseCase{MRPRepo: &fakeMRPRepo{suggestion: baseSuggestion()}, PlannedRepo: planR, Auth: &fakeAuth{canCreate: true}}
	result, err := uc.Execute(context.Background(), 42)
	if err != nil || result.PlannedCode != 77 {
		t.Fatalf("idempotent conversion failed: result=%+v err=%v", result, err)
	}
	if planR.created != nil {
		t.Fatal("existing suggestion origin must not create a duplicate planned order")
	}
}

func TestAutoRelease_OnlyEligibleInterFactorySuggestion(t *testing.T) {
	suggestion := baseSuggestion()
	suggestion.InterFactory = true
	suggestion.AutoRelease = true
	planR := &fakePlannedRepo{nextNum: 9000}
	firmer := &fakeFirmer{}
	uc := &FirmarSugestaoMRPUseCase{MRPRepo: &fakeMRPRepo{suggestion: suggestion}, PlannedRepo: planR, Auth: &fakeAuth{canCreate: true}, Firmer: firmer}
	if err := uc.ExecuteAutoRelease(context.Background(), suggestion.PlanCode); err != nil {
		t.Fatalf("auto release failed: %v", err)
	}
	if planR.created == nil || !firmer.called {
		t.Fatal("eligible inter-factory suggestion was not converted and firmed")
	}
}

func TestFirmarSugestao_RejectsUnauthorized(t *testing.T) {
	uc := &FirmarSugestaoMRPUseCase{
		MRPRepo:     &fakeMRPRepo{suggestion: baseSuggestion()},
		PlannedRepo: &fakePlannedRepo{nextNum: 1},
		Auth:        &fakeAuth{canCreate: false},
	}
	_, err := uc.Execute(context.Background(), 42)
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestFirmarSugestao_SuggestionNotFound(t *testing.T) {
	uc := &FirmarSugestaoMRPUseCase{
		MRPRepo:     &fakeMRPRepo{errGet: errors.New("not found")},
		PlannedRepo: &fakePlannedRepo{nextNum: 1},
		Auth:        &fakeAuth{canCreate: true},
	}
	_, err := uc.Execute(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error for missing suggestion, got nil")
	}
}

func TestFirmarSugestao_PlannedRepoError(t *testing.T) {
	uc := &FirmarSugestaoMRPUseCase{
		MRPRepo:     &fakeMRPRepo{suggestion: baseSuggestion()},
		PlannedRepo: &fakePlannedRepo{nextNum: 1, errCreate: errors.New("db error")},
		Auth:        &fakeAuth{canCreate: true},
	}
	_, err := uc.Execute(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error from planned repo, got nil")
	}
}

func TestFirmarSugestao_ParentItemCodeAddedToNotes(t *testing.T) {
	sugg := baseSuggestion()
	parent := int64(5000)
	sugg.ParentItemCode = &parent

	planR := &fakePlannedRepo{nextNum: 1}
	uc := &FirmarSugestaoMRPUseCase{
		MRPRepo:     &fakeMRPRepo{suggestion: sugg},
		PlannedRepo: planR,
		Auth:        &fakeAuth{canCreate: true},
	}
	_, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planR.created.Notes == nil {
		t.Fatal("notes should not be nil when parent_item_code is set")
	}
}
