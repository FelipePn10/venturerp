package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

type RestrictionReasonRepository interface {
	Create(ctx context.Context, r *entity.RestrictionReason) (*entity.RestrictionReason, error)
	GetByCode(ctx context.Context, code int64) (*entity.RestrictionReason, error)
	List(ctx context.Context) ([]*entity.RestrictionReason, error)
	Update(ctx context.Context, r *entity.RestrictionReason) (*entity.RestrictionReason, error)
	Delete(ctx context.Context, code int64) error
}
