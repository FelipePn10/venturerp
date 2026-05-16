package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

type SalesOrderRepository interface {
	NextOrderNumber(ctx context.Context, enterpriseCode int64) (int64, error)
	Create(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error)
	Update(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error)
	GetByCode(ctx context.Context, code int64) (*entity.SalesOrder, error)
	List(ctx context.Context) ([]*entity.SalesOrder, error)
	ListByCustomer(ctx context.Context, customerCode int64) ([]*entity.SalesOrder, error)
	ListByStatus(ctx context.Context, status entity.SalesOrderStatus) ([]*entity.SalesOrder, error)
	ListByDateRange(ctx context.Context, from, to time.Time) ([]*entity.SalesOrder, error)
	Cancel(ctx context.Context, code int64) error
	Block(ctx context.Context, code int64, reason string) error
	Unblock(ctx context.Context, code int64) error
	ChangeStatus(ctx context.Context, code int64, status entity.SalesOrderStatus) error

	CreateItem(ctx context.Context, item *entity.SalesOrderItem) (*entity.SalesOrderItem, error)
	UpdateItem(ctx context.Context, item *entity.SalesOrderItem) (*entity.SalesOrderItem, error)
	ListItems(ctx context.Context, salesOrderCode int64) ([]*entity.SalesOrderItem, error)
	CancelItem(ctx context.Context, itemCode int64) error
}
