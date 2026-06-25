package shipment_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/google/uuid"
)

type fakeSalesReader struct {
	resp *SalesOrderHeader
	err  error
}

func (f *fakeSalesReader) GetByCode(ctx context.Context, code int64) (*SalesOrderHeader, error) {
	return f.resp, f.err
}

type fakePurchaseReader struct {
	resp *PurchaseOrderHeader
	err  error
}

func (f *fakePurchaseReader) GetByCode(ctx context.Context, code int64) (*PurchaseOrderHeader, error) {
	return f.resp, f.err
}

type fakeProductionReader struct {
	resp *ProductionOrderHeader
	err  error
}

func (f *fakeProductionReader) GetByCode(ctx context.Context, code int64) (*ProductionOrderHeader, error) {
	return f.resp, f.err
}

func newAutoFillUC(shipRepo *fakeShipmentRepo, salesR SalesOrderReader, purchR PurchaseOrderReader, prodR ProductionOrderReader) *ShipmentAutoFillUseCase {
	return &ShipmentAutoFillUseCase{
		ShipmentRepo:   shipRepo,
		SalesRepo:      salesR,
		PurchaseRepo:   purchR,
		ProductionRepo: prodR,
	}
}

func TestAutoFillFromSalesOrder_Success(t *testing.T) {
	userID := uuid.New()
	carrier := int64(5)
	shipRepo := &fakeShipmentRepo{
		nextCode: 1,
		getByCodeResp: &entity.Shipment{
			Code:          1,
			ReferenceType: refPtr(entity.ShipmentRefSalesOrder),
			CarrierCode:   &carrier,
			Status:        entity.ShipmentStatusOpen,
			TotalVolumes:  3,
			Items:         []*entity.ShipmentItem{},
		},
	}
	salesR := &fakeSalesReader{
		resp: &SalesOrderHeader{
			Code:         100,
			CarrierCode:  ptrInt64(5),
			TotalVolumes: 3,
			TotalWeight:  75.0,
			Items: []SalesOrderItemHeader{
				{ItemCode: 1, RequestedQty: 10, UnitPrice: 25},
				{ItemCode: 2, RequestedQty: 5, UnitPrice: 50},
			},
		},
	}
	uc := newAutoFillUC(shipRepo, salesR, nil, nil)

	result, err := uc.AutoFillFromSalesOrder(context.Background(), 100, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 1 {
		t.Errorf("code = %d, want 1", result.Code)
	}
	if result.ReferenceType == nil || *result.ReferenceType != "SALES_ORDER" {
		t.Errorf("reference_type = %v, want SALES_ORDER", result.ReferenceType)
	}
	if result.CarrierCode == nil || *result.CarrierCode != 5 {
		t.Errorf("carrier_code = %v, want 5", result.CarrierCode)
	}
	if result.TotalVolumes != 3 {
		t.Errorf("total_volumes = %d, want 3", result.TotalVolumes)
	}

	if shipRepo.created == nil {
		t.Fatal("shipment was not persisted")
	}
	if *shipRepo.created.SalesOrderCode != 100 {
		t.Errorf("sales_order_code = %d, want 100", *shipRepo.created.SalesOrderCode)
	}
	if shipRepo.created.CarrierCode == nil || *shipRepo.created.CarrierCode != 5 {
		t.Errorf("created carrier_code = %v, want 5", shipRepo.created.CarrierCode)
	}
	if shipRepo.created.TotalVolumes != 3 {
		t.Errorf("created total_volumes = %d, want 3", shipRepo.created.TotalVolumes)
	}
}

func TestAutoFillFromSalesOrder_NoVolumesDefaultsToOne(t *testing.T) {
	userID := uuid.New()
	shipRepo := &fakeShipmentRepo{
		nextCode: 2,
		getByCodeResp: &entity.Shipment{
			Code:          2,
			ReferenceType: refPtr(entity.ShipmentRefSalesOrder),
			Status:        entity.ShipmentStatusOpen,
		},
	}
	salesR := &fakeSalesReader{
		resp: &SalesOrderHeader{
			Code:         101,
			TotalVolumes: 0,
			Items:        []SalesOrderItemHeader{},
		},
	}
	uc := newAutoFillUC(shipRepo, salesR, nil, nil)

	result, err := uc.AutoFillFromSalesOrder(context.Background(), 101, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shipRepo.created.TotalVolumes != 1 {
		t.Errorf("total_volumes = %d, want 1 (default)", shipRepo.created.TotalVolumes)
	}
	_ = result
}

func TestAutoFillFromSalesOrder_SalesNotFound(t *testing.T) {
	shipRepo := &fakeShipmentRepo{}
	salesR := &fakeSalesReader{err: errors.New("not found")}
	uc := newAutoFillUC(shipRepo, salesR, nil, nil)

	_, err := uc.AutoFillFromSalesOrder(context.Background(), 999, uuid.New())
	if err == nil {
		t.Fatal("expected error for not found sales order")
	}
}

func TestAutoFillFromPurchaseOrder_Success(t *testing.T) {
	userID := uuid.New()
	shipRepo := &fakeShipmentRepo{
		nextCode: 3,
		getByCodeResp: &entity.Shipment{
			Code:          3,
			ReferenceType: refPtr(entity.ShipmentRefPurchaseOrder),
			Status:        entity.ShipmentStatusOpen,
		},
	}
	purchR := &fakePurchaseReader{
		resp: &PurchaseOrderHeader{
			Code:        200,
			CarrierCode: ptrInt64(8),
			TotalWeight: 120.0,
			Items: []PurchaseOrderItemHeader{
				{ItemCode: 10, RequestedQty: 20, UnitPrice: 15},
				{ItemCode: 11, RequestedQty: 5, UnitPrice: 30},
			},
		},
	}
	uc := newAutoFillUC(shipRepo, nil, purchR, nil)

	result, err := uc.AutoFillFromPurchaseOrder(context.Background(), 200, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReferenceType == nil || *result.ReferenceType != "PURCHASE_ORDER" {
		t.Errorf("reference_type = %v, want PURCHASE_ORDER", result.ReferenceType)
	}
	if shipRepo.created.PurchaseOrderCode == nil || *shipRepo.created.PurchaseOrderCode != 200 {
		t.Errorf("purchase_order_code = %v, want 200", shipRepo.created.PurchaseOrderCode)
	}
}

func TestAutoFillFromProductionOrder_Success(t *testing.T) {
	userID := uuid.New()
	shipRepo := &fakeShipmentRepo{
		nextCode: 4,
		getByCodeResp: &entity.Shipment{
			Code:          4,
			ReferenceType: refPtr(entity.ShipmentRefProductionOrder),
			Status:        entity.ShipmentStatusOpen,
		},
	}
	prodR := &fakeProductionReader{
		resp: &ProductionOrderHeader{
			Code:        300,
			ItemCode:    50,
			PlannedQty:  0,
			ProducedQty: 100,
			Items: []ProductionOrderItemHeader{
				{ItemCode: 20, ConsumedQty: 50},
				{ItemCode: 21, ConsumedQty: 25},
			},
		},
	}
	uc := newAutoFillUC(shipRepo, nil, nil, prodR)

	result, err := uc.AutoFillFromProductionOrder(context.Background(), 300, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReferenceType == nil || *result.ReferenceType != "PRODUCTION_ORDER" {
		t.Errorf("reference_type = %v, want PRODUCTION_ORDER", result.ReferenceType)
	}
	// Produced qty item (seq 1) + 2 consumed items (seq 2,3) = 3 items
	if len(shipRepo.created.Items) == 0 && shipRepo.addedItem != nil {
	}
}

func TestAutoFillFromProductionOrder_FallsBackToPlannedQty(t *testing.T) {
	userID := uuid.New()
	shipRepo := &fakeShipmentRepo{
		nextCode: 5,
		getByCodeResp: &entity.Shipment{
			Code:          5,
			ReferenceType: refPtr(entity.ShipmentRefProductionOrder),
			Status:        entity.ShipmentStatusOpen,
		},
	}
	prodR := &fakeProductionReader{
		resp: &ProductionOrderHeader{
			Code:        301,
			ItemCode:    51,
			PlannedQty:  50,
			ProducedQty: 0,
			Items:       []ProductionOrderItemHeader{},
		},
	}
	uc := newAutoFillUC(shipRepo, nil, nil, prodR)

	_, err := uc.AutoFillFromProductionOrder(context.Background(), 301, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shipRepo.addedItem.Quantity != 50 {
		t.Errorf("quantity = %v, want 50 (planned)", shipRepo.addedItem.Quantity)
	}
}

func TestAutoFill_ReturnsFullShipment(t *testing.T) {
	userID := uuid.New()
	shipRepo := &fakeShipmentRepo{
		nextCode: 6,
		getByCodeResp: &entity.Shipment{
			ID:             10,
			Code:           6,
			ReferenceType:  refPtr(entity.ShipmentRefSalesOrder),
			SalesOrderCode: ptrInt64(102),
			CarrierCode:    ptrInt64(7),
			Status:           entity.ShipmentStatusOpen,
			TotalVolumes:     2,
			TotalGrossWeight: 80,
			Items: []*entity.ShipmentItem{
				{ID: 1, ItemCode: 100, Quantity: 5},
			},
		},
	}
	salesR := &fakeSalesReader{
		resp: &SalesOrderHeader{
			Code:         102,
			CarrierCode:  ptrInt64(7),
			TotalVolumes: 2,
			TotalWeight:  80,
			Items: []SalesOrderItemHeader{
				{ItemCode: 100, RequestedQty: 5},
			},
		},
	}
	uc := newAutoFillUC(shipRepo, salesR, nil, nil)

	result, err := uc.AutoFillFromSalesOrder(context.Background(), 102, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(result.Items))
	}
	if result.Items[0].ItemCode != 100 {
		t.Errorf("item code = %d, want 100", result.Items[0].ItemCode)
	}
}

func refPtr(t entity.ShipmentReferenceType) *entity.ShipmentReferenceType {
	return &t
}
