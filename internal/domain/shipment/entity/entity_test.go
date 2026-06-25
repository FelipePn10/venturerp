package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShipmentReferenceTypeConstants(t *testing.T) {
	if ShipmentRefSalesOrder != "SALES_ORDER" {
		t.Errorf("SalesOrder = %q, want SALES_ORDER", ShipmentRefSalesOrder)
	}
	if ShipmentRefPurchaseOrder != "PURCHASE_ORDER" {
		t.Errorf("PurchaseOrder = %q, want PURCHASE_ORDER", ShipmentRefPurchaseOrder)
	}
	if ShipmentRefProductionOrder != "PRODUCTION_ORDER" {
		t.Errorf("ProductionOrder = %q, want PRODUCTION_ORDER", ShipmentRefProductionOrder)
	}
}

func TestShipmentStatusConstants(t *testing.T) {
	if ShipmentStatusOpen != "OPEN" {
		t.Errorf("Open = %q, want OPEN", ShipmentStatusOpen)
	}
	if ShipmentStatusSeparated != "SEPARATED" {
		t.Errorf("Separated = %q, want SEPARATED", ShipmentStatusSeparated)
	}
	if ShipmentStatusConferred != "CONFERRED" {
		t.Errorf("Conferred = %q, want CONFERRED", ShipmentStatusConferred)
	}
	if ShipmentStatusShipped != "SHIPPED" {
		t.Errorf("Shipped = %q, want SHIPPED", ShipmentStatusShipped)
	}
	if ShipmentStatusCancelled != "CANCELLED" {
		t.Errorf("Cancelled = %q, want CANCELLED", ShipmentStatusCancelled)
	}
}

func TestShipmentReferenceTypePointer(t *testing.T) {
	ref := ShipmentRefSalesOrder
	ptr := &ref
	if *ptr != ShipmentRefSalesOrder {
		t.Fatalf("pointer ref = %q, want SALES_ORDER", *ptr)
	}
}

func TestShipmentStructDefaults(t *testing.T) {
	s := &Shipment{
		Code:             1,
		Status:           ShipmentStatusOpen,
		TotalVolumes:     0,
		TotalGrossWeight: 0,
		CreatedBy:        uuid.New(),
	}
	if s.Items != nil {
		t.Error("Items should be nil by default")
	}
	if s.ReferenceType != nil {
		t.Error("ReferenceType should be nil by default")
	}
	if s.SalesOrderCode != nil {
		t.Error("SalesOrderCode should be nil by default")
	}
	if s.PurchaseOrderCode != nil {
		t.Error("PurchaseOrderCode should be nil by default")
	}
	if s.ProductionOrderCode != nil {
		t.Error("ProductionOrderCode should be nil by default")
	}
}

func TestShipmentAllReferenceTypes(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	tests := []struct {
		name       string
		refType    ShipmentReferenceType
		setSales    bool
		setPurchase  bool
		setProduction bool
	}{
		{"sales order", ShipmentRefSalesOrder, true, false, false},
		{"purchase order", ShipmentRefPurchaseOrder, false, true, false},
		{"production order", ShipmentRefProductionOrder, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := tt.refType
			s := &Shipment{
				Code:          100,
				ReferenceType: &ref,
				Status:        ShipmentStatusOpen,
				CreatedAt:     now,
				UpdatedAt:     now,
				CreatedBy:     userID,
			}

			code := int64(5000)
			if tt.setSales {
				s.SalesOrderCode = &code
			}
			if tt.setPurchase {
				s.PurchaseOrderCode = &code
			}
			if tt.setProduction {
				s.ProductionOrderCode = &code
			}

			if s.ReferenceType == nil {
				t.Fatal("ReferenceType should not be nil")
			}
			if *s.ReferenceType != tt.refType {
				t.Errorf("ReferenceType = %q, want %q", *s.ReferenceType, tt.refType)
			}
		})
	}
}

func TestShipmentItemDefaults(t *testing.T) {
	it := &ShipmentItem{
		Sequence:  1,
		ItemCode:  100,
		Quantity:  10,
	}
	if it.IsConferred {
		t.Error("IsConferred should be false by default")
	}
	if it.ConferredQty != 0 {
		t.Errorf("ConferredQty = %v, want 0", it.ConferredQty)
	}
}
