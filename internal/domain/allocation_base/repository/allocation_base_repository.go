package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
)

type AllocationBaseRepository interface {
	Create(ctx context.Context, ab *entity.AllocationBase) (*entity.AllocationBase, error)
	AddItem(ctx context.Context, item *entity.AllocationBaseItem) (*entity.AllocationBaseItem, error)
	GetByCode(ctx context.Context, code int32) (*entity.AllocationBase, error)
	GetItems(ctx context.Context, baseCode int32) ([]*entity.AllocationBaseItem, error)
	List(ctx context.Context) ([]*entity.AllocationBase, error)
	Delete(ctx context.Context, code int32) error
	DeleteItems(ctx context.Context, baseCode int32) error
}
