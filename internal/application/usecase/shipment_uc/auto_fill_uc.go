package shipment_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/google/uuid"
)

type SalesOrderReader interface {
	GetByCode(ctx context.Context, code int64) (*SalesOrderHeader, error)
}

type SalesOrderHeader struct {
	Code         int64
	CarrierCode  *int64
	CustomerCode *int64
	TotalVolumes int
	TotalWeight  float64
	TotalGross   float64
	TotalNet     float64
	Items        []SalesOrderItemHeader
}

type SalesOrderItemHeader struct {
	ItemCode        int64
	RequestedQty    float64
	UnitPrice       float64
	TotalGross      float64
	TotalNet        float64
	IPIPct          float64
	ICMSPct         float64
	PISPct          float64
	COFINSPct       float64
	STPct           float64
	UnitWeightNet   float64
	UnitWeightGross float64
}

type PurchaseOrderReader interface {
	GetByCode(ctx context.Context, code int64) (*PurchaseOrderHeader, error)
}

type PurchaseOrderHeader struct {
	Code         int64
	CarrierCode  *int64
	SupplierCode *int64
	TotalWeight  float64
	Items        []PurchaseOrderItemHeader
}

type PurchaseOrderItemHeader struct {
	ItemCode     int64
	RequestedQty float64
	UnitPrice    float64
	TotalPrice   float64
	IPIPct       float64
	ICMSPct      float64
	ICMSSTPct    float64
}

type ProductionOrderReader interface {
	GetByCode(ctx context.Context, code int64) (*ProductionOrderHeader, error)
}

type ProductionOrderHeader struct {
	Code        int64
	ItemCode    int64
	PlannedQty  float64
	ProducedQty float64
	Items       []ProductionOrderItemHeader
}

type ProductionOrderItemHeader struct {
	ItemCode    int64
	ConsumedQty float64
}

type ShipmentAutoFillUseCase struct {
	ShipmentRepo   repository.ShipmentRepository
	SalesRepo      SalesOrderReader
	PurchaseRepo   PurchaseOrderReader
	ProductionRepo ProductionOrderReader
}

func (uc *ShipmentAutoFillUseCase) AutoFillFromSalesOrder(ctx context.Context, salesOrderCode int64, createdBy uuid.UUID) (*response.ShipmentResponse, error) {
	so, err := uc.SalesRepo.GetByCode(ctx, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("sales order %d: %w", salesOrderCode, err)
	}

	code, err := uc.ShipmentRepo.NextCode(ctx)
	if err != nil {
		return nil, err
	}

	volumes := so.TotalVolumes
	if volumes == 0 {
		volumes = 1
	}

	refType := entity.ShipmentRefSalesOrder
	s := &entity.Shipment{
		Code:             code,
		ReferenceType:    &refType,
		SalesOrderCode:   &salesOrderCode,
		CarrierCode:      so.CarrierCode,
		Status:           entity.ShipmentStatusOpen,
		TotalVolumes:     volumes,
		TotalGrossWeight: so.TotalWeight,
		CreatedBy:        createdBy,
	}

	created, err := uc.ShipmentRepo.Create(ctx, s)
	if err != nil {
		return nil, err
	}

	for i, it := range so.Items {
		item := &entity.ShipmentItem{
			ShipmentID:      created.ID,
			Sequence:        i + 1,
			ItemCode:        it.ItemCode,
			Quantity:        it.RequestedQty,
			UnitNetWeight:   it.UnitWeightNet,
			UnitGrossWeight: it.UnitWeightGross,
		}
		if _, err := uc.ShipmentRepo.AddItem(ctx, item); err != nil {
			return nil, fmt.Errorf("adding auto-fill item %d: %w", it.ItemCode, err)
		}
	}
	_ = uc.ShipmentRepo.RecalcTotals(ctx, code)

	return uc.getShipment(ctx, code)
}

func (uc *ShipmentAutoFillUseCase) AutoFillFromPurchaseOrder(ctx context.Context, purchaseOrderCode int64, createdBy uuid.UUID) (*response.ShipmentResponse, error) {
	po, err := uc.PurchaseRepo.GetByCode(ctx, purchaseOrderCode)
	if err != nil {
		return nil, fmt.Errorf("purchase order %d: %w", purchaseOrderCode, err)
	}

	code, err := uc.ShipmentRepo.NextCode(ctx)
	if err != nil {
		return nil, err
	}

	refType := entity.ShipmentRefPurchaseOrder
	s := &entity.Shipment{
		Code:              code,
		ReferenceType:     &refType,
		PurchaseOrderCode: &purchaseOrderCode,
		CarrierCode:       po.CarrierCode,
		Status:            entity.ShipmentStatusOpen,
		TotalVolumes:      1,
		TotalGrossWeight:  po.TotalWeight,
		CreatedBy:         createdBy,
	}

	created, err := uc.ShipmentRepo.Create(ctx, s)
	if err != nil {
		return nil, err
	}

	for i, it := range po.Items {
		item := &entity.ShipmentItem{
			ShipmentID: created.ID,
			Sequence:   i + 1,
			ItemCode:   it.ItemCode,
			Quantity:   it.RequestedQty,
		}
		if _, err := uc.ShipmentRepo.AddItem(ctx, item); err != nil {
			return nil, fmt.Errorf("adding auto-fill item %d: %w", it.ItemCode, err)
		}
	}

	return uc.getShipment(ctx, code)
}

func (uc *ShipmentAutoFillUseCase) AutoFillFromProductionOrder(ctx context.Context, productionOrderCode int64, createdBy uuid.UUID) (*response.ShipmentResponse, error) {
	po, err := uc.ProductionRepo.GetByCode(ctx, productionOrderCode)
	if err != nil {
		return nil, fmt.Errorf("production order %d: %w", productionOrderCode, err)
	}

	code, err := uc.ShipmentRepo.NextCode(ctx)
	if err != nil {
		return nil, err
	}

	qty := po.ProducedQty
	if qty == 0 {
		qty = po.PlannedQty
	}

	refType := entity.ShipmentRefProductionOrder
	s := &entity.Shipment{
		Code:                code,
		ReferenceType:       &refType,
		ProductionOrderCode: &productionOrderCode,
		Status:              entity.ShipmentStatusOpen,
		TotalVolumes:        1,
		CreatedBy:           createdBy,
	}

	created, err := uc.ShipmentRepo.Create(ctx, s)
	if err != nil {
		return nil, err
	}

	seq := 1
	if qty > 0 {
		item := &entity.ShipmentItem{
			ShipmentID: created.ID,
			Sequence:   seq,
			ItemCode:   po.ItemCode,
			Quantity:   qty,
		}
		if _, err := uc.ShipmentRepo.AddItem(ctx, item); err != nil {
			return nil, fmt.Errorf("adding auto-fill produced item %d: %w", po.ItemCode, err)
		}
		seq++
	}

	for _, it := range po.Items {
		item := &entity.ShipmentItem{
			ShipmentID: created.ID,
			Sequence:   seq,
			ItemCode:   it.ItemCode,
			Quantity:   it.ConsumedQty,
		}
		if _, err := uc.ShipmentRepo.AddItem(ctx, item); err != nil {
			return nil, fmt.Errorf("adding auto-fill consumed item %d: %w", it.ItemCode, err)
		}
		seq++
	}

	return uc.getShipment(ctx, code)
}

func (uc *ShipmentAutoFillUseCase) getShipment(ctx context.Context, code int64) (*response.ShipmentResponse, error) {
	s, err := uc.ShipmentRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(s), nil
}
