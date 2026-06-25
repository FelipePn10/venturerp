package shipment_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/google/uuid"
)

type fakeShipmentRepo struct {
	repository.ShipmentRepository
	nextCode    int64
	nextCodeErr error
	created     *entity.Shipment
	createErr   error

	getByCodeResp *entity.Shipment
	getByCodeErr  error

	listResp []*entity.Shipment
	listErr  error

	listBySalesResp []*entity.Shipment
	listByPOErr     error
	listByPResp     []*entity.Shipment
	listByProcErr   error

	statusUpdated   bool
	lastStatus      entity.ShipmentStatus
	updateStatusErr error

	addedItem  *entity.ShipmentItem
	addItemErr error

	listItemsResp []*entity.ShipmentItem
	listItemsErr  error

	conferredItemID int64
	conferredQty    float64
	conferItemErr   error
}

func (f *fakeShipmentRepo) NextCode(ctx context.Context) (int64, error) {
	return f.nextCode, f.nextCodeErr
}

func (f *fakeShipmentRepo) Create(ctx context.Context, s *entity.Shipment) (*entity.Shipment, error) {
	f.created = s
	if f.createErr != nil {
		return nil, f.createErr
	}
	s.ID = 1
	return s, nil
}

func (f *fakeShipmentRepo) GetByCode(ctx context.Context, code int64) (*entity.Shipment, error) {
	if f.getByCodeErr != nil {
		return nil, f.getByCodeErr
	}
	return f.getByCodeResp, nil
}

func (f *fakeShipmentRepo) List(ctx context.Context) ([]*entity.Shipment, error) {
	return f.listResp, f.listErr
}

func (f *fakeShipmentRepo) ListFiltered(ctx context.Context, _ repository.ShipmentFilter) ([]*entity.Shipment, error) {
	return f.listResp, f.listErr
}

func (f *fakeShipmentRepo) ListBySalesOrder(ctx context.Context, code int64) ([]*entity.Shipment, error) {
	return f.listBySalesResp, f.listByPOErr
}

func (f *fakeShipmentRepo) ListByPurchaseOrder(ctx context.Context, code int64) ([]*entity.Shipment, error) {
	return f.listBySalesResp, f.listByPOErr
}

func (f *fakeShipmentRepo) ListByProductionOrder(ctx context.Context, code int64) ([]*entity.Shipment, error) {
	return f.listByPResp, f.listByProcErr
}

func (f *fakeShipmentRepo) ListByReference(ctx context.Context, refType entity.ShipmentReferenceType, refCode int64) ([]*entity.Shipment, error) {
	return f.listBySalesResp, f.listByPOErr
}

func (f *fakeShipmentRepo) UpdateStatus(ctx context.Context, code int64, status entity.ShipmentStatus, _ *uuid.UUID, _ string) error {
	if f.updateStatusErr != nil {
		return f.updateStatusErr
	}
	f.statusUpdated = true
	f.lastStatus = status
	return nil
}

func (f *fakeShipmentRepo) RecalcTotals(ctx context.Context, code int64) error { return nil }

func (f *fakeShipmentRepo) AddItem(ctx context.Context, item *entity.ShipmentItem) (*entity.ShipmentItem, error) {
	f.addedItem = item
	if f.addItemErr != nil {
		return nil, f.addItemErr
	}
	item.ID = 1
	return item, nil
}

func (f *fakeShipmentRepo) ListItems(ctx context.Context, shipmentID int64) ([]*entity.ShipmentItem, error) {
	return f.listItemsResp, f.listItemsErr
}

func (f *fakeShipmentRepo) ConferItem(ctx context.Context, itemID int64, conferredQty float64) error {
	if f.conferItemErr != nil {
		return f.conferItemErr
	}
	f.conferredItemID = itemID
	f.conferredQty = conferredQty
	return nil
}

func TestCreateShipment(t *testing.T) {
	repo := &fakeShipmentRepo{nextCode: 1001}
	uc := &ShipmentUseCase{Repo: repo}

	ref := entity.ShipmentRefSalesOrder
	soCode := int64(42)
	result, err := uc.Create(context.Background(), CreateShipmentInput{
		ReferenceType:    &ref,
		SalesOrderCode:   &soCode,
		CarrierCode:      ptrInt64(10),
		TotalVolumes:     5,
		TotalGrossWeight: 150.5,
		Notes:            strPtr("urgente"),
		CreatedBy:        uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 1001 {
		t.Errorf("code = %d, want 1001", result.Code)
	}
	if result.ReferenceType == nil || *result.ReferenceType != "SALES_ORDER" {
		t.Errorf("reference_type = %v, want SALES_ORDER", result.ReferenceType)
	}
	if result.Status != "OPEN" {
		t.Errorf("status = %s, want OPEN", result.Status)
	}
	if repo.created == nil {
		t.Fatal("shipment was not persisted")
	}
}

func TestCreateShipment_RepoError(t *testing.T) {
	repo := &fakeShipmentRepo{nextCode: 1, createErr: errors.New("db down")}
	uc := &ShipmentUseCase{Repo: repo}

	_, err := uc.Create(context.Background(), CreateShipmentInput{CreatedBy: uuid.New()})
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestAddItem_ToOpenShipment(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{ID: 1, Status: entity.ShipmentStatusOpen},
	}
	uc := &ShipmentUseCase{Repo: repo}

	result, err := uc.AddItem(context.Background(), AddShipmentItemInput{
		ShipmentCode: 1001,
		Sequence:     1,
		ItemCode:     500,
		Quantity:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemCode != 500 {
		t.Errorf("item_code = %d, want 500", result.ItemCode)
	}
}

func TestAddItem_ToShippedFails(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{ID: 1, Status: entity.ShipmentStatusShipped},
	}
	uc := &ShipmentUseCase{Repo: repo}

	_, err := uc.AddItem(context.Background(), AddShipmentItemInput{
		ShipmentCode: 1001, Sequence: 1, ItemCode: 500, Quantity: 10,
	})
	if err == nil {
		t.Fatal("expected error when adding item to shipped shipment")
	}
}

func TestGet(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{Code: 2001, Status: entity.ShipmentStatusOpen},
	}
	uc := &ShipmentUseCase{Repo: repo}

	result, err := uc.Get(context.Background(), 2001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 2001 {
		t.Errorf("code = %d, want 2001", result.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := &fakeShipmentRepo{getByCodeErr: errors.New("not found")}
	uc := &ShipmentUseCase{Repo: repo}

	_, err := uc.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestList(t *testing.T) {
	s1 := &entity.Shipment{Code: 1, Status: entity.ShipmentStatusOpen}
	s2 := &entity.Shipment{Code: 2, Status: entity.ShipmentStatusShipped}
	repo := &fakeShipmentRepo{listResp: []*entity.Shipment{s1, s2}}
	uc := &ShipmentUseCase{Repo: repo}

	result, err := uc.List(context.Background(), repository.ShipmentFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestConferItem(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{Code: 1001, Status: entity.ShipmentStatusSeparated},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.ConferItem(context.Background(), 1001, 10, 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.conferredItemID != 10 || repo.conferredQty != 5 {
		t.Errorf("conferred item %d qty %v, want 10 / 5", repo.conferredItemID, repo.conferredQty)
	}
}

func TestSeparate_ReservesAndTransitions(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{
			ID: 1, Code: 1001, Status: entity.ShipmentStatusOpen,
			Items: []*entity.ShipmentItem{{ItemCode: 1, Quantity: 5}},
		},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.Separate(context.Background(), 1001, uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastStatus != entity.ShipmentStatusSeparated {
		t.Errorf("lastStatus = %s, want SEPARATED", repo.lastStatus)
	}
}

func TestSeparate_InvalidTransition(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{Code: 1001, Status: entity.ShipmentStatusShipped},
	}
	uc := &ShipmentUseCase{Repo: repo}
	if err := uc.Separate(context.Background(), 1001, uuid.New()); err == nil {
		t.Fatal("expected invalid-transition error from SHIPPED")
	}
}

func TestConfer(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{
			Code: 1001, Status: entity.ShipmentStatusSeparated,
			Items: []*entity.ShipmentItem{{ItemCode: 1, IsConferred: true}},
		},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.Confer(context.Background(), 1001, uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.statusUpdated || repo.lastStatus != entity.ShipmentStatusConferred {
		t.Errorf("statusUpdated=%v lastStatus=%s, want true / CONFERRED", repo.statusUpdated, repo.lastStatus)
	}
}

func TestShip_AllConferredNoDivergence(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{
			Code:   1001,
			Status: entity.ShipmentStatusConferred,
			Items: []*entity.ShipmentItem{
				{ItemCode: 1, Quantity: 3, ConferredQty: 3, IsConferred: true},
				{ItemCode: 2, Quantity: 5, ConferredQty: 5, IsConferred: true},
			},
		},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.Ship(context.Background(), 1001, uuid.New(), false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastStatus != entity.ShipmentStatusShipped {
		t.Errorf("lastStatus=%s, want SHIPPED", repo.lastStatus)
	}
}

func TestShip_NotAllConferred(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{
			Code:   1001,
			Status: entity.ShipmentStatusConferred,
			Items: []*entity.ShipmentItem{
				{ItemCode: 1, IsConferred: true},
				{ItemCode: 2, IsConferred: false},
			},
		},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.Ship(context.Background(), 1001, uuid.New(), false); err == nil {
		t.Fatal("expected error when not all items are conferred")
	}
}

func TestShip_DivergenceBlockedUnlessAccepted(t *testing.T) {
	mk := func() *fakeShipmentRepo {
		return &fakeShipmentRepo{
			getByCodeResp: &entity.Shipment{
				Code:   1001,
				Status: entity.ShipmentStatusConferred,
				Items: []*entity.ShipmentItem{
					{ItemCode: 1, Quantity: 10, ConferredQty: 8, IsConferred: true}, // falta
				},
			},
		}
	}
	uc := &ShipmentUseCase{Repo: mk()}
	if err := uc.Ship(context.Background(), 1001, uuid.New(), false); err == nil {
		t.Fatal("expected divergence to block shipping")
	}
	uc2 := &ShipmentUseCase{Repo: mk()}
	if err := uc2.Ship(context.Background(), 1001, uuid.New(), true); err != nil {
		t.Fatalf("accepting divergence should ship: %v", err)
	}
}

func TestCancel(t *testing.T) {
	repo := &fakeShipmentRepo{
		getByCodeResp: &entity.Shipment{Code: 1001, Status: entity.ShipmentStatusOpen},
	}
	uc := &ShipmentUseCase{Repo: repo}

	if err := uc.Cancel(context.Background(), 1001, uuid.New(), "cliente cancelou"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.statusUpdated || repo.lastStatus != entity.ShipmentStatusCancelled {
		t.Errorf("statusUpdated=%v lastStatus=%s, want true / CANCELLED", repo.statusUpdated, repo.lastStatus)
	}
}

func TestListBySalesOrder(t *testing.T) {
	s1 := &entity.Shipment{Code: 1, SalesOrderCode: ptrInt64(500), Status: entity.ShipmentStatusOpen}
	repo := &fakeShipmentRepo{listBySalesResp: []*entity.Shipment{s1}}
	uc := &ShipmentUseCase{Repo: repo}

	result, err := uc.ListBySalesOrder(context.Background(), 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

func ptrInt64(v int64) *int64 { return &v }

func strPtr(s string) *string { return &s }
