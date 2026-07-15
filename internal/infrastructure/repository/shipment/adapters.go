package shipment

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	purchaseentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	purchaserepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type SalesOrderAdapter struct {
	Repo salesrepo.SalesOrderRepository
}

func (a *SalesOrderAdapter) GetByCode(ctx context.Context, code int64) (*shipment_uc.SalesOrderHeader, error) {
	so, err := a.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	items, err := a.Repo.ListItems(ctx, code)
	if err != nil {
		return nil, err
	}
	h := &shipment_uc.SalesOrderHeader{
		Code:         so.Code,
		CarrierCode:  so.CarrierCode,
		CustomerCode: so.CustomerCode,
		TotalVolumes: int(so.VolumeQuantity),
		TotalWeight:  so.TotalWeightGross,
		TotalGross:   so.TotalGross,
		TotalNet:     so.TotalNet,
	}
	for _, it := range items {
		h.Items = append(h.Items, shipment_uc.SalesOrderItemHeader{
			ItemCode:        it.ItemCode,
			RequestedQty:    it.RequestedQty,
			UnitPrice:       it.UnitPrice,
			TotalGross:      it.TotalGross,
			TotalNet:        it.TotalNet,
			IPIPct:          it.IPIPct,
			ICMSPct:         it.ICMSPct,
			PISPct:          it.PISPct,
			COFINSPct:       it.COFINSPct,
			STPct:           it.STPct,
			UnitWeightNet:   it.UnitWeightNet,
			UnitWeightGross: it.UnitWeightGross,
		})
	}
	return h, nil
}

var _ shipment_uc.SalesOrderReader = (*SalesOrderAdapter)(nil)

type PurchaseOrderAdapter struct {
	Repo purchaserepo.PurchaseOrderRepository
}

func (a *PurchaseOrderAdapter) GetByCode(ctx context.Context, code int64) (*shipment_uc.PurchaseOrderHeader, error) {
	po, err := a.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	h := &shipment_uc.PurchaseOrderHeader{
		Code:         po.Code,
		CarrierCode:  po.CarrierCode,
		SupplierCode: po.SupplierCode,
		TotalWeight:  po.FreightValue,
	}
	for _, it := range po.Items {
		h.Items = append(h.Items, shipment_uc.PurchaseOrderItemHeader{
			ItemCode:     it.ItemCode,
			RequestedQty: it.RequestedQty,
			UnitPrice:    it.UnitPrice,
			TotalPrice:   it.TotalPrice,
			IPIPct:       it.IPIPct,
			ICMSPct:      it.ICMSPct,
			ICMSSTPct:    it.ICMSSTPct,
		})
	}
	return h, nil
}

var _ shipment_uc.PurchaseOrderReader = (*PurchaseOrderAdapter)(nil)

type ProductionOrderAdapter struct {
	Repo productionrepo.ProductionOrderRepository
}

func (a *ProductionOrderAdapter) GetByCode(ctx context.Context, code int64) (*shipment_uc.ProductionOrderHeader, error) {
	po, err := a.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	h := &shipment_uc.ProductionOrderHeader{
		Code:        po.ID,
		ItemCode:    po.ItemCode,
		PlannedQty:  po.PlannedQty,
		ProducedQty: po.ProducedQty,
	}
	return h, nil
}

var _ shipment_uc.ProductionOrderReader = (*ProductionOrderAdapter)(nil)

func init() {
	_ = (*SalesOrderAdapter)(nil)
	_ = (*PurchaseOrderAdapter)(nil)
	_ = (*ProductionOrderAdapter)(nil)
}

var _ = []interface{}{
	(*salesentity.SalesOrder)(nil),
	(*purchaseentity.PurchaseOrder)(nil),
	(*entity.ProductionOrder)(nil),
}
