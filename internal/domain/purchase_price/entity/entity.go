package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PurchasePriceTable struct {
	ID            int64
	Code          int64
	Description   string
	CurrencyCode  string
	ValidityStart *time.Time
	ValidityEnd   *time.Time
	IsActive      bool
	CreatedAt     time.Time
	CreatedBy     uuid.UUID
	UpdatedAt     time.Time
	Items         []*PurchasePriceTableItem
}

func NewPurchasePriceTable(code int64, description, currency string, createdBy uuid.UUID) (*PurchasePriceTable, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if currency == "" {
		currency = "BRL"
	}
	now := time.Now()
	return &PurchasePriceTable{
		Code:         code,
		Description:  description,
		CurrencyCode: currency,
		IsActive:     true,
		CreatedAt:    now,
		CreatedBy:    createdBy,
		UpdatedAt:    now,
	}, nil
}

type PurchasePriceTableItem struct {
	ID           int64
	TableID      int64
	ItemCode     int64
	SupplierCode *int64
	UOM          *string
	Price        float64
	MinQty       float64
	IsActive     bool
	CreatedAt    time.Time
}

func NewPurchasePriceTableItem(tableID, itemCode int64, price float64) (*PurchasePriceTableItem, error) {
	if tableID == 0 || itemCode == 0 {
		return nil, fmt.Errorf("table_id and item_code are required")
	}
	if price < 0 {
		return nil, fmt.Errorf("price must not be negative")
	}
	return &PurchasePriceTableItem{
		TableID:   tableID,
		ItemCode:  itemCode,
		Price:     price,
		IsActive:  true,
		CreatedAt: time.Now(),
	}, nil
}
