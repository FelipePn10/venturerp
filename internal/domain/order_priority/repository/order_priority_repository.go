package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
)

type OrderPriorityRepository interface {
	Create(ctx context.Context, op *entity.OrderPriority) (*entity.OrderPriority, error)
	Update(ctx context.Context, op *entity.OrderPriority) (*entity.OrderPriority, error)
	GetByCode(ctx context.Context, code int64) (*entity.OrderPriority, error)
	FindByValue(ctx context.Context, value float64) (*entity.OrderPriority, error)
	List(ctx context.Context) ([]*entity.OrderPriority, error)
	Delete(ctx context.Context, code int64) error
}
