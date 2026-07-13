package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
)

type SourceFilter struct {
	EnterpriseID int64
	SupplierCode *int64
	TableCode    *int64
	Start, End   time.Time
	Source       string
}

type ApplySourceSelection struct {
	SourceType string
	SourceID   int64
}

type PurchasePriceRepository interface {
	CreateTable(context.Context, *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error)
	UpdateTable(context.Context, *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error)
	GetTableByCode(ctx context.Context, enterpriseID, code int64) (*entity.PurchasePriceTable, error)
	ListTables(ctx context.Context, enterpriseID int64, supplierCode *int64, onlyActive bool) ([]*entity.PurchasePriceTable, error)
	NextTableCode(ctx context.Context, enterpriseID int64) (int64, error)
	AddItem(context.Context, int64, *entity.PurchasePriceTableItem) (*entity.PurchasePriceTableItem, error)
	ListItems(ctx context.Context, enterpriseID, tableID int64) ([]*entity.PurchasePriceTableItem, error)
	DeleteItem(ctx context.Context, enterpriseID, id int64) error
	GetItemPrice(ctx context.Context, enterpriseID, tableCode, itemCode int64, supplierCode *int64) (*entity.PurchasePriceTableItem, error)
	IsPreferredSupplier(ctx context.Context, enterpriseID, itemCode, supplierCode int64) (bool, error)
	ListItemCandidates(ctx context.Context, enterpriseID, tableCode int64, mode, order string, classificationID *int64) ([]entity.ItemCandidate, error)
	ReplaceAdjustments(ctx context.Context, enterpriseID, priceItemID int64, adjustments []*entity.PriceAdjustment) error
	CopyAdjustments(ctx context.Context, enterpriseID, sourceItemID, targetItemID int64, mode string) error
	ListSourcePrices(ctx context.Context, filter SourceFilter) ([]entity.SourcePrice, error)
	ApplySourcePrices(ctx context.Context, enterpriseID, tableCode int64, overwrite bool, selections []ApplySourceSelection) (int64, error)
}
