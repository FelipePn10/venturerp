package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
)

type PurchaseOrderRepository interface {
	NextOrderNumber(ctx context.Context, enterpriseCode int64) (int64, error)
	Create(ctx context.Context, o *entity.PurchaseOrder) (*entity.PurchaseOrder, error)
	// CreateWithItems atomically creates a purchase order and its items in a
	// single transaction (used by the MRP suggestion approval).
	CreateWithItems(ctx context.Context, o *entity.PurchaseOrder, items []*entity.PurchaseOrderItem) (*entity.PurchaseOrder, error)
	Update(ctx context.Context, o *entity.PurchaseOrder) (*entity.PurchaseOrder, error)
	GetByCode(ctx context.Context, code int64) (*entity.PurchaseOrder, error)
	List(ctx context.Context) ([]*entity.PurchaseOrder, error)
	Cancel(ctx context.Context, code int64) error

	CreateItem(ctx context.Context, item *entity.PurchaseOrderItem) (*entity.PurchaseOrderItem, error)
	UpdateItem(ctx context.Context, item *entity.PurchaseOrderItem) (*entity.PurchaseOrderItem, error)
	ListItems(ctx context.Context, purchaseOrderCode int64) ([]*entity.PurchaseOrderItem, error)
	CancelItem(ctx context.Context, itemCode int64) error

	ListBySupplier(ctx context.Context, supplierCode int64) ([]*entity.PurchaseOrder, error)
	ListByStatus(ctx context.Context, status entity.PurchaseOrderStatus) ([]*entity.PurchaseOrder, error)
}
