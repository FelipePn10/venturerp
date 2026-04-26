package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
)

type AllocationBaseRepository interface {
	Create(ctx context.Context, ab *entity.AllocationBase) (*entity.AllocationBase, error)
	AddItem(ctx context.Context, item *entity.AllocationBaseItem) (*entity.AllocationBaseItem, error)
	GetByID(ctx context.Context, id int64) (*entity.AllocationBase, error)
	GetItems(ctx context.Context, baseID int64) ([]*entity.AllocationBaseItem, error)
	List(ctx context.Context) ([]*entity.AllocationBase, error)
	Delete(ctx context.Context, id int64) error
	DeleteItems(ctx context.Context, baseID int64) error
}
