package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

type RestrictionRepository interface {
	Create(ctx context.Context, r *entity.Restriction) (*entity.Restriction, error)
	Update(ctx context.Context, r *entity.Restriction) (*entity.Restriction, error)
	GetByCode(ctx context.Context, code int64) (*entity.Restriction, error)
	GetByItemCode(ctx context.Context, itemCode int64) ([]*entity.Restriction, error)
	List(ctx context.Context) ([]*entity.Restriction, error)
	Deactivate(ctx context.Context, code int64) error
	// ListRestrictedItemCodes returns item codes that have at least one ACTIVE restriction
	// for any of the given item codes. Used by MRP to skip restricted items.
	ListRestrictedItemCodes(ctx context.Context, itemCodes []int64) (map[int64]struct{}, error)
	AddDominant(ctx context.Context, dominant *entity.RestrictionDominant) (*entity.RestrictionDominant, error)
	AddDeterminant(ctx context.Context, det *entity.RestrictionDeterminant) (*entity.RestrictionDeterminant, error)
	DeleteDominant(ctx context.Context, id int64) error
	DeleteDeterminant(ctx context.Context, id int64) error
	GetDominants(ctx context.Context, restrictionID int64) ([]*entity.RestrictionDominant, error)
	GetDeterminants(ctx context.Context, restrictionID int64) ([]*entity.RestrictionDeterminant, error)
}
