package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
)

type PurchaseQuotationRepository interface {
	Create(ctx context.Context, q *entity.PurchaseQuotation) (*entity.PurchaseQuotation, error)
	GetByCode(ctx context.Context, code int64) (*entity.PurchaseQuotation, error)
	List(ctx context.Context, onlyOpen bool) ([]*entity.PurchaseQuotation, error)
	NextCode(ctx context.Context) (int64, error)
	UpdateStatus(ctx context.Context, code int64, status string) error

	AddItem(ctx context.Context, item *entity.PurchaseQuotationItem) (*entity.PurchaseQuotationItem, error)
	ListItems(ctx context.Context, quotationCode int64) ([]*entity.PurchaseQuotationItem, error)

	AddSupplier(ctx context.Context, s *entity.PurchaseQuotationSupplier) (*entity.PurchaseQuotationSupplier, error)
	ListSuppliers(ctx context.Context, quotationCode int64) ([]*entity.PurchaseQuotationSupplier, error)

	UpsertPrice(ctx context.Context, p *entity.PurchaseQuotationPrice) (*entity.PurchaseQuotationPrice, error)
	ListPricesByItem(ctx context.Context, quotationItemID int64) ([]*entity.PurchaseQuotationPrice, error)
	ListSelectedPrices(ctx context.Context, quotationCode int64) ([]*entity.PurchaseQuotationPrice, error)
	// SelectPrice marks one price as selected and clears the others for the item.
	SelectPrice(ctx context.Context, priceID int64) (*entity.PurchaseQuotationPrice, error)
	GetItem(ctx context.Context, itemID int64) (*entity.PurchaseQuotationItem, error)
}
