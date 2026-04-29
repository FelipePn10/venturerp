package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type DeliveryRescheduleRepository interface {
	Create(ctx context.Context, r *entity.DeliveryReschedule) (*entity.DeliveryReschedule, error)
	GetByCode(ctx context.Context, code int64) (*entity.DeliveryReschedule, error)
	ListByOrder(ctx context.Context, orderCode int64) ([]*entity.DeliveryReschedule, error)
	ListByItem(ctx context.Context, itemCode valueobject.ItemCode) ([]*entity.DeliveryReschedule, error)
	Delete(ctx context.Context, code int64) error
}
