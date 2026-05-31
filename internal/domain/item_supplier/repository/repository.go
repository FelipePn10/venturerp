package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
)

type ItemSupplierRepository interface {
	Upsert(ctx context.Context, s *entity.ItemPreferredSupplier) (*entity.ItemPreferredSupplier, error)
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemPreferredSupplier, error)
	// GetPreferred returns the lowest-ranking active supplier for the item.
	GetPreferred(ctx context.Context, itemCode int64) (*entity.ItemPreferredSupplier, error)
	Delete(ctx context.Context, id int64) error
}
