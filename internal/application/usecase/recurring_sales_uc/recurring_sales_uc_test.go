package recurring_sales_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	rsrepo "github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	representativeentity "github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	representativerepo "github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
	"github.com/google/uuid"
)

type rsAllowAuth struct{ ports.AuthService }

func (rsAllowAuth) CanCreateSalesOrder(context.Context) bool      { return true }
func (rsAllowAuth) UserID(context.Context) (uuid.UUID, error)     { return uuid.New(), nil }
func (rsAllowAuth) EnterpriseCode(context.Context) (int64, error) { return 1, nil }

type activeRepresentativeRepo struct {
	representativerepo.RepresentativeRepository
}

func (activeRepresentativeRepo) Get(_ context.Context, code int64) (*representativeentity.Representative, error) {
	return &representativeentity.Representative{Code: code, IsActive: true}, nil
}

type blockedRecurringRepresentativeRepo struct {
	representativerepo.RepresentativeRepository
}

func (blockedRecurringRepresentativeRepo) Get(_ context.Context, code int64) (*representativeentity.Representative, error) {
	return &representativeentity.Representative{Code: code, IsActive: true, Blocked: true}, nil
}

type fakeRSRepo struct {
	rows   []*entity.RecurringSale
	params *entity.Parameters
}

func (f *fakeRSRepo) UpsertParameters(_ context.Context, p *entity.Parameters) (*entity.Parameters, error) {
	return p, nil
}
func (f *fakeRSRepo) GetParameters(context.Context, int64) (*entity.Parameters, error) {
	if f.params != nil {
		return f.params, nil
	}
	return &entity.Parameters{CurrentMonthBillingLimitDay: 10, IndefiniteDeliveryDay: 10, FixedTermDeliveryDay: 10}, nil
}
func (f *fakeRSRepo) CreateAdjustmentDate(_ context.Context, v *entity.AdjustmentDate) (*entity.AdjustmentDate, error) {
	return v, nil
}
func (f *fakeRSRepo) ListAdjustmentDates(context.Context, rsrepo.Filter) ([]*entity.AdjustmentDate, error) {
	return nil, nil
}
func (f *fakeRSRepo) Create(_ context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	v.Code = int64(len(f.rows) + 1)
	f.rows = append(f.rows, v)
	return v, nil
}
func (f *fakeRSRepo) CreateWithRepresentatives(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	return f.Create(ctx, v)
}
func (f *fakeRSRepo) Update(_ context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error) {
	return v, nil
}
func (f *fakeRSRepo) Get(context.Context, int64) (*entity.RecurringSale, error) {
	return f.rows[0], nil
}
func (f *fakeRSRepo) List(context.Context, rsrepo.Filter) ([]*entity.RecurringSale, error) {
	return f.rows, nil
}
func (f *fakeRSRepo) AddRepresentative(_ context.Context, v *entity.Representative) (*entity.Representative, error) {
	return v, nil
}
func (f *fakeRSRepo) MarkOrderGenerated(_ context.Context, code int64, orderCode int64) (*entity.RecurringSale, error) {
	f.rows[0].GeneratedOrderCode = &orderCode
	return f.rows[0], nil
}
func (f *fakeRSRepo) ClearGeneratedOrder(context.Context, int64) (*entity.RecurringSale, error) {
	return f.rows[0], nil
}
func (f *fakeRSRepo) Deactivate(context.Context, int64, *string) (*entity.RecurringSale, error) {
	f.rows[0].IsActive = false
	return f.rows[0], nil
}
func (f *fakeRSRepo) CreateAdjustmentLink(context.Context, int64, int64) error { return nil }
func (f *fakeRSRepo) ReserveOperation(_ context.Context, operation *entity.Operation) (*entity.Operation, bool, error) {
	return operation, false, nil
}
func (f *fakeRSRepo) CompleteOperation(context.Context, *entity.Operation, int64) error { return nil }
func (f *fakeRSRepo) FailOperation(context.Context, *entity.Operation) error            { return nil }

type fakeOrderCreator struct {
	dto request.CreateSalesOrderDTO
}

func (f *fakeOrderCreator) Execute(_ context.Context, dto request.CreateSalesOrderDTO) (*response.SalesOrderResponse, error) {
	f.dto = dto
	return &response.SalesOrderResponse{Code: 88, EmissionDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}, nil
}

type fakeOrderItemCreator struct {
	dtos []request.CreateSalesOrderItemDTO
}

func (f *fakeOrderItemCreator) Execute(_ context.Context, dto request.CreateSalesOrderItemDTO) (*response.SalesOrderItemResponse, error) {
	f.dtos = append(f.dtos, dto)
	return &response.SalesOrderItemResponse{Code: int64(len(f.dtos)), SalesOrderCode: dto.SalesOrderCode}, nil
}

func TestCreateRequiresAdjustmentDateForIndefiniteTerm(t *testing.T) {
	uc := &UseCase{Repo: &fakeRSRepo{}, Auth: rsAllowAuth{}}
	_, err := uc.Create(context.Background(), request.CreateRecurringSaleDTO{
		EnterpriseCode: 1, CustomerCode: 2, ItemCode: 3, SaleDate: "2026-07-01",
		Representatives: []request.CreateRecurringSaleRepresentativeDTO{{RepresentativeCode: 9, IsPrimary: true}},
		CreatedBy:       uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected missing next_adjustment_date validation")
	}
}

func TestCreateFixedTermRequiresPaymentsAfterGrace(t *testing.T) {
	payments, months, grace, value := 2, 12, 2, 100.0
	uc := &UseCase{Repo: &fakeRSRepo{}, Auth: rsAllowAuth{}}
	_, err := uc.Create(context.Background(), request.CreateRecurringSaleDTO{
		EnterpriseCode: 1, CustomerCode: 2, ItemCode: 3, SaleDate: "2026-07-01", TermType: "FIXED",
		MonthsQuantity: &months, PaymentsQuantity: &payments, GraceMonths: grace, PaymentValue: &value,
		Representatives: []request.CreateRecurringSaleRepresentativeDTO{{RepresentativeCode: 9, IsPrimary: true}},
		CreatedBy:       uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected payments/grace validation")
	}
}

func TestCreateRequiresExactlyOnePrimaryRepresentative(t *testing.T) {
	uc := &UseCase{Repo: &fakeRSRepo{}, Auth: rsAllowAuth{}}
	_, err := uc.Create(context.Background(), request.CreateRecurringSaleDTO{
		EnterpriseCode: 1, CustomerCode: 2, ItemCode: 3, SaleDate: "2026-07-01", NextAdjustmentDate: "2027-07-01",
		Representatives: []request.CreateRecurringSaleRepresentativeDTO{{RepresentativeCode: 9}, {RepresentativeCode: 10}},
		CreatedBy:       uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected primary representative validation")
	}
}

func TestGenerateSalesOrderCreatesMonthlyLinesUntilAdjustment(t *testing.T) {
	next := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	orderCreator := &fakeOrderCreator{}
	itemCreator := &fakeOrderItemCreator{}
	repo := &fakeRSRepo{
		params: &entity.Parameters{CurrentMonthBillingLimitDay: 10, IndefiniteDeliveryDay: 31, FixedTermDeliveryDay: 15, GroupOrderItemTotal: true},
		rows: []*entity.RecurringSale{{
			Code: 7, EnterpriseCode: 1, CustomerCode: 2, ItemCode: 3, MovementType: entity.MovementSale,
			TermType: entity.TermIndefinite, SaleDate: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
			NextAdjustmentDate: &next, Quantity: 2, UnitValue: 50, CreatedBy: uuid.Nil, IsActive: true,
			Representatives: []*entity.Representative{{RepresentativeCode: 9, IsPrimary: true, CommissionPercent: 5}},
		}},
	}
	uc := &UseCase{Repo: repo, Auth: rsAllowAuth{}, SalesOrders: orderCreator, SalesOrderItems: itemCreator, Representatives: activeRepresentativeRepo{}}

	got, err := uc.GenerateSalesOrder(context.Background(), 7, request.MarkRecurringSaleOrderDTO{IdempotencyKey: "test-order-2026-07", Competence: "2026-07"})
	if err != nil {
		t.Fatalf("GenerateSalesOrder() error = %v", err)
	}
	if got.GeneratedOrderCode == nil || *got.GeneratedOrderCode != 88 {
		t.Fatalf("generated_order_code = %v, want 88", got.GeneratedOrderCode)
	}
	if orderCreator.dto.CustomerCode == nil || *orderCreator.dto.CustomerCode != 2 {
		t.Fatalf("order customer not mapped: %+v", orderCreator.dto.CustomerCode)
	}
	if orderCreator.dto.RepresentativeCode == nil || *orderCreator.dto.RepresentativeCode != 9 {
		t.Fatalf("primary representative not mapped: %+v", orderCreator.dto.RepresentativeCode)
	}
	if len(itemCreator.dtos) != 3 {
		t.Fatalf("generated lines = %d, want Jul/Aug/Sep", len(itemCreator.dtos))
	}
	if itemCreator.dtos[0].RequestedQty != 1 || itemCreator.dtos[0].UnitPrice != 100 {
		t.Fatalf("grouped first line qty/unit = %.2f/%.2f, want 1/100", itemCreator.dtos[0].RequestedQty, itemCreator.dtos[0].UnitPrice)
	}
	if itemCreator.dtos[1].DeliveryDate == nil || *itemCreator.dtos[1].DeliveryDate != "2026-08-31" {
		t.Fatalf("second delivery date = %v, want 2026-08-31", itemCreator.dtos[1].DeliveryDate)
	}
}

func TestGenerateSalesOrderKeepsManualLinkCompatibility(t *testing.T) {
	repo := &fakeRSRepo{rows: []*entity.RecurringSale{{Code: 7}}}
	uc := &UseCase{Repo: repo, Auth: rsAllowAuth{}}

	got, err := uc.GenerateSalesOrder(context.Background(), 7, request.MarkRecurringSaleOrderDTO{OrderCode: 123})
	if err != nil {
		t.Fatalf("GenerateSalesOrder() manual link error = %v", err)
	}
	if got.GeneratedOrderCode == nil || *got.GeneratedOrderCode != 123 {
		t.Fatalf("generated_order_code = %v, want manual code 123", got.GeneratedOrderCode)
	}
}

func TestRecurringCreationAndGenerationRejectBlockedRepresentative(t *testing.T) {
	next := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeRSRepo{rows: []*entity.RecurringSale{{Code: 7, MovementType: entity.MovementSale, TermType: entity.TermIndefinite, NextAdjustmentDate: &next, Representatives: []*entity.Representative{{RepresentativeCode: 9, IsPrimary: true}}}}}
	uc := &UseCase{Repo: repo, Auth: rsAllowAuth{}, Representatives: blockedRecurringRepresentativeRepo{}, SalesOrders: &fakeOrderCreator{}, SalesOrderItems: &fakeOrderItemCreator{}}
	_, err := uc.Create(context.Background(), request.CreateRecurringSaleDTO{EnterpriseCode: 1, CustomerCode: 2, ItemCode: 3, SaleDate: "2026-07-01", NextAdjustmentDate: "2026-10-01", Representatives: []request.CreateRecurringSaleRepresentativeDTO{{RepresentativeCode: 9, IsPrimary: true}}})
	if err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("criação deveria rejeitar representante bloqueado: %v", err)
	}
	_, err = uc.GenerateSalesOrder(context.Background(), 7, request.MarkRecurringSaleOrderDTO{})
	if err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("geração deveria rejeitar representante bloqueado: %v", err)
	}
}

func TestAdjustmentPreviewExplainsPreviousAndNewValues(t *testing.T) {
	adjustmentDate := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	orderCode := int64(99)
	index := "IPCA"
	repo := &fakeRSRepo{rows: []*entity.RecurringSale{{
		Code: 7, EnterpriseCode: 1, CustomerCode: 2, ItemCode: 3,
		MovementType: entity.MovementSale, TermType: entity.TermIndefinite,
		SaleDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), NextAdjustmentDate: &adjustmentDate,
		Quantity: 2, UnitValue: 50, GeneratedOrderCode: &orderCode, IsActive: true, AdjustmentIndex: &index,
	}}}
	uc := &UseCase{Repo: repo, Auth: rsAllowAuth{}}
	got, err := uc.CalculateAdjustment(context.Background(), request.CalculateRecurringSalesAdjustmentDTO{
		AdjustmentDate: "2027-07-01", AdjustmentPercent: 10, Reason: "Reajuste anual",
		LegalBasis: "Cláusula 8", Confirm: false,
	})
	if err != nil {
		t.Fatalf("CalculateAdjustment() error = %v", err)
	}
	if len(got.Impacts) != 1 {
		t.Fatalf("impactos = %d, esperado 1", len(got.Impacts))
	}
	impact := got.Impacts[0]
	if impact.PreviousUnitValue != 50 || impact.NewUnitValue != 55 || impact.PreviousTotal != 100 || impact.NewTotal != 110 {
		t.Fatalf("valores inesperados: %+v", impact)
	}
	if impact.AdjustmentIndex != "IPCA" || impact.LegalBasis != "Cláusula 8" || impact.Reason != "Reajuste anual" {
		t.Fatalf("explicação incompleta: %+v", impact)
	}
}

func TestActiveRecurringSaleCanBeCancelledBeforeFirstOrder(t *testing.T) {
	got := toRecurringSaleResponse(&entity.RecurringSale{IsActive: true, LifecycleStatus: entity.LifecycleActive})
	if !got.CanCancel {
		t.Fatal("recorrência ativa deve permitir cancelamento antes do primeiro pedido")
	}
}
