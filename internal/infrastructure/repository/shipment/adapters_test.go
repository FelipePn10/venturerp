package shipment

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	purchaseentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	purchaserepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type fakeSalesOrderRepo struct {
	salesrepo.SalesOrderRepository
	resp *salesentity.SalesOrder
	err  error
}

func (f *fakeSalesOrderRepo) GetByCode(ctx context.Context, code int64) (*salesentity.SalesOrder, error) {
	return f.resp, f.err
}

type fakePurchaseOrderRepo struct {
	purchaserepo.PurchaseOrderRepository
	resp *purchaseentity.PurchaseOrder
	err  error
}

func (f *fakePurchaseOrderRepo) GetByCode(ctx context.Context, code int64) (*purchaseentity.PurchaseOrder, error) {
	return f.resp, f.err
}

type fakeProductionOrderRepo struct {
	productionrepo.ProductionOrderRepository
	resp *productionentity.ProductionOrder
	err  error
}

func (f *fakeProductionOrderRepo) GetByCode(ctx context.Context, id int64) (*productionentity.ProductionOrder, error) {
	return f.resp, f.err
}

func ptrFl64(v float64) *float64     { return &v }
func ptrStr(v string) *string        { return &v }
func ptrTime(v time.Time) *time.Time { return &v }

func TestSalesOrderAdapter(t *testing.T) {
	delivery := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	fake := &fakeSalesOrderRepo{
		resp: &salesentity.SalesOrder{
			Code:             42,
			TotalGross:       1000.00,
			TotalNet:         850.00,
			TotalWeightGross: 120.0,
			VolumeQuantity:   3,
			Items: []*salesentity.SalesOrderItem{
				{
					ItemCode:        100,
					RequestedQty:    10,
					UnitPrice:       50,
					TotalGross:      500,
					TotalNet:        425,
					IPIPct:          5,
					ICMSPct:         18,
					PISPct:          1.65,
					COFINSPct:       7.6,
					STPct:           0,
					UnitWeightNet:   2.5,
					UnitWeightGross: 3.0,
					DeliveryDate:    &delivery,
				},
			},
		},
	}
	a := &SalesOrderAdapter{Repo: fake}

	h, err := a.GetByCode(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Code != 42 {
		t.Errorf("code = %d, want 42", h.Code)
	}
	if h.TotalGross != 1000.0 {
		t.Errorf("total_gross = %v, want 1000", h.TotalGross)
	}
	if h.TotalNet != 850.0 {
		t.Errorf("total_net = %v, want 850", h.TotalNet)
	}
	if h.TotalVolumes != 3 {
		t.Errorf("total_volumes = %d, want 3", h.TotalVolumes)
	}
	if h.TotalWeight != 120.0 {
		t.Errorf("total_weight = %v, want 120", h.TotalWeight)
	}
	if len(h.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(h.Items))
	}
	it := h.Items[0]
	if it.ItemCode != 100 {
		t.Errorf("item_code = %d, want 100", it.ItemCode)
	}
	if it.RequestedQty != 10 {
		t.Errorf("requested_qty = %v, want 10", it.RequestedQty)
	}
	if it.UnitPrice != 50 {
		t.Errorf("unit_price = %v, want 50", it.UnitPrice)
	}
	if it.IPIPct != 5 {
		t.Errorf("ipi_pct = %v, want 5", it.IPIPct)
	}
	if it.ICMSPct != 18 {
		t.Errorf("icms_pct = %v, want 18", it.ICMSPct)
	}
	if it.PISPct != 1.65 {
		t.Errorf("pis_pct = %v, want 1.65", it.PISPct)
	}
	if it.UnitWeightNet != 2.5 {
		t.Errorf("weight_net = %v, want 2.5", it.UnitWeightNet)
	}
	if it.UnitWeightGross != 3.0 {
		t.Errorf("weight_gross = %v, want 3.0", it.UnitWeightGross)
	}
}

func TestSalesOrderAdapter_ImplementsInterface(t *testing.T) {
	var _ shipment_uc.SalesOrderReader = (*SalesOrderAdapter)(nil)
}

func TestPurchaseOrderAdapter(t *testing.T) {
	fake := &fakePurchaseOrderRepo{
		resp: &purchaseentity.PurchaseOrder{
			Code:         55,
			FreightValue: 200.0,
			Items: []*purchaseentity.PurchaseOrderItem{
				{
					ItemCode:     500,
					RequestedQty: 100,
					UnitPrice:    15.50,
					TotalPrice:   1550.00,
					IPIPct:       4,
					ICMSPct:      12,
					ICMSSTPct:    0,
				},
			},
		},
	}
	a := &PurchaseOrderAdapter{Repo: fake}

	h, err := a.GetByCode(context.Background(), 55)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Code != 55 {
		t.Errorf("code = %d, want 55", h.Code)
	}
	if len(h.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(h.Items))
	}
	it := h.Items[0]
	if it.ItemCode != 500 {
		t.Errorf("item_code = %d, want 500", it.ItemCode)
	}
	if it.RequestedQty != 100 {
		t.Errorf("requested_qty = %v, want 100", it.RequestedQty)
	}
	if it.UnitPrice != 15.50 {
		t.Errorf("unit_price = %v, want 15.50", it.UnitPrice)
	}
	if it.TotalPrice != 1550.00 {
		t.Errorf("total_price = %v, want 1550", it.TotalPrice)
	}
	if it.IPIPct != 4 {
		t.Errorf("ipi_pct = %v, want 4", it.IPIPct)
	}
	if it.ICMSPct != 12 {
		t.Errorf("icms_pct = %v, want 12", it.ICMSPct)
	}
}

func TestPurchaseOrderAdapter_ImplementsInterface(t *testing.T) {
	var _ shipment_uc.PurchaseOrderReader = (*PurchaseOrderAdapter)(nil)
}

func TestProductionOrderAdapter(t *testing.T) {
	fake := &fakeProductionOrderRepo{
		resp: &productionentity.ProductionOrder{
			ID:          99,
			ItemCode:    700,
			PlannedQty:  50,
			ProducedQty: 48,
		},
	}
	a := &ProductionOrderAdapter{Repo: fake}

	h, err := a.GetByCode(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Code != 99 {
		t.Errorf("code = %d, want 99", h.Code)
	}
	if h.ItemCode != 700 {
		t.Errorf("item_code = %d, want 700", h.ItemCode)
	}
	if h.PlannedQty != 50 {
		t.Errorf("planned_qty = %v, want 50", h.PlannedQty)
	}
	if h.ProducedQty != 48 {
		t.Errorf("produced_qty = %v, want 48", h.ProducedQty)
	}
}

func TestProductionOrderAdapter_ImplementsInterface(t *testing.T) {
	var _ shipment_uc.ProductionOrderReader = (*ProductionOrderAdapter)(nil)
}
