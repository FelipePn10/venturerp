package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
)

type PurchasePriceRepository interface {
	CreateTable(ctx context.Context, t *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error)
	UpdateTable(ctx context.Context, t *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error)
	GetTableByCode(ctx context.Context, code int64) (*entity.PurchasePriceTable, error)
	ListTables(ctx context.Context, onlyActive bool) ([]*entity.PurchasePriceTable, error)
	NextTableCode(ctx context.Context) (int64, error)

	AddItem(ctx context.Context, item *entity.PurchasePriceTableItem) (*entity.PurchasePriceTableItem, error)
	ListItems(ctx context.Context, tableID int64) ([]*entity.PurchasePriceTableItem, error)
	DeleteItem(ctx context.Context, id int64) error
	// GetItemPrice resolves the price for an item in a table by code. Prefers a
	// supplier-specific row when supplierCode is given, falling back to the
	// supplier-agnostic row.
	GetItemPrice(ctx context.Context, tableCode, itemCode int64, supplierCode *int64) (*entity.PurchasePriceTableItem, error)
}
