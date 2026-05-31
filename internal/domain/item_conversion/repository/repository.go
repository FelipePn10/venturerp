package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
)

type ItemConversionRepository interface {
	Create(ctx context.Context, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error)
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error)
	// Get returns the direct conversion (item, from, to) if registered.
	Get(ctx context.Context, itemCode int64, fromUOM, toUOM string) (*entity.ItemUnitConversion, error)
	Delete(ctx context.Context, id int64) error
}
