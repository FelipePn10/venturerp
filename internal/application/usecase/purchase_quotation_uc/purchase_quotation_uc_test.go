package purchase_quotation_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	qrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository"
	reqentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeQuotRepo struct {
	qrepo.PurchaseQuotationRepository
	nextCode  int64
	created   *entity.PurchaseQuotation
	errCode   error
	errCreate error
}

func (r *fakeQuotRepo) NextCode(_ context.Context) (int64, error) {
	return r.nextCode, r.errCode
}
func (r *fakeQuotRepo) Create(_ context.Context, q *entity.PurchaseQuotation) (*entity.PurchaseQuotation, error) {
	if r.errCreate != nil {
		return nil, r.errCreate
	}
	r.created = q
	return q, nil
}
func (r *fakeQuotRepo) AddItem(_ context.Context, item *entity.PurchaseQuotationItem) (*entity.PurchaseQuotationItem, error) {
	item.ID = 1
	return item, nil
}
func (r *fakeQuotRepo) GetByCode(_ context.Context, code int64) (*entity.PurchaseQuotation, error) {
	if r.created != nil && r.created.Code == code {
		return r.created, nil
	}
	return nil, errors.New("not found")
}
func (r *fakeQuotRepo) ListItems(_ context.Context, _ int64) ([]*entity.PurchaseQuotationItem, error) {
	return nil, nil
}
func (r *fakeQuotRepo) ListSuppliers(_ context.Context, _ int64) ([]*entity.PurchaseQuotationSupplier, error) {
	return nil, nil
}
func (r *fakeQuotRepo) List(_ context.Context, _ bool) ([]*entity.PurchaseQuotation, error) {
	if r.created != nil {
		return []*entity.PurchaseQuotation{r.created}, nil
	}
	return nil, nil
}

type fakeReqRepo struct {
	reqrepo.PurchaseRequisitionRepository
}

type fakePlannedRepoQ struct {
	plannedrepo.PlannedOrderRepository
}

func newUC(repo *fakeQuotRepo) *PurchaseQuotationUseCase {
	return NewPurchaseQuotationUseCase(repo, &fakeReqRepo{}, &fakePlannedRepoQ{})
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreateQuotation_Success(t *testing.T) {
	repo := &fakeQuotRepo{nextCode: 5001}
	uc := newUC(repo)

	dto := request.CreatePurchaseQuotationDTO{
		EnterpriseCode: 1,
		CreatedBy:      uuid.New(),
	}
	result, err := uc.Create(context.Background(), dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 5001 {
		t.Errorf("Code = %d, want 5001", result.Code)
	}
	if result.Status != string(entity.QuotationOpen) {
		t.Errorf("Status = %q, want %q", result.Status, entity.QuotationOpen)
	}
}

func TestCreateQuotation_NoEnterpriseCode(t *testing.T) {
	repo := &fakeQuotRepo{nextCode: 1}
	uc := newUC(repo)

	_, err := uc.Create(context.Background(), request.CreatePurchaseQuotationDTO{
		EnterpriseCode: 0,
		CreatedBy:      uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error for missing enterprise_code, got nil")
	}
}

func TestCreateQuotation_CodeGenError(t *testing.T) {
	repo := &fakeQuotRepo{errCode: errors.New("seq error")}
	uc := newUC(repo)

	_, err := uc.Create(context.Background(), request.CreatePurchaseQuotationDTO{
		EnterpriseCode: 1,
		CreatedBy:      uuid.New(),
	})
	if err == nil {
		t.Fatal("expected code generation error, got nil")
	}
}

func TestCreateQuotation_RepoError(t *testing.T) {
	repo := &fakeQuotRepo{nextCode: 1, errCreate: errors.New("db down")}
	uc := newUC(repo)

	_, err := uc.Create(context.Background(), request.CreatePurchaseQuotationDTO{
		EnterpriseCode: 1,
		CreatedBy:      uuid.New(),
	})
	if err == nil {
		t.Fatal("expected create error, got nil")
	}
}

func TestGetQuotation_NotFound(t *testing.T) {
	repo := &fakeQuotRepo{nextCode: 1}
	uc := newUC(repo)

	_, err := uc.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestListQuotations_ReturnsCreated(t *testing.T) {
	repo := &fakeQuotRepo{nextCode: 1}
	uc := newUC(repo)
	// Create first
	_, _ = uc.Create(context.Background(), request.CreatePurchaseQuotationDTO{
		EnterpriseCode: 1, CreatedBy: uuid.New(),
	})

	list, err := uc.List(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one quotation in list")
	}
}

// ── Entity-level test ─────────────────────────────────────────────────────────

func TestNewPurchaseQuotation_Validation(t *testing.T) {
	_, err := entity.NewPurchaseQuotation(0, 1, uuid.New())
	if err == nil {
		t.Fatal("code=0 must fail")
	}
	_, err = entity.NewPurchaseQuotation(1, 0, uuid.New())
	if err == nil {
		t.Fatal("enterprise_code=0 must fail")
	}
	q, err := entity.NewPurchaseQuotation(100, 1, uuid.New())
	if err != nil {
		t.Fatalf("valid args should succeed: %v", err)
	}
	if q.Status != entity.QuotationOpen {
		t.Errorf("new quotation must be OPEN, got %q", q.Status)
	}
}

// ── ReqItem Balance helper ────────────────────────────────────────────────────

func TestRequisitionItemBalance(t *testing.T) {
	item := &reqentity.PurchaseRequisitionItem{
		Quantity:    100,
		AttendedQty: 40,
	}
	if item.Balance() != 60 {
		t.Errorf("Balance = %v, want 60", item.Balance())
	}
}
