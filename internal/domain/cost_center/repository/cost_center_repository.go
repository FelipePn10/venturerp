package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
)

type CostCenterRepository interface {
	Create(ctx context.Context, cc *entity.CostCenter) (*entity.CostCenter, error)
	Update(ctx context.Context, cc *entity.CostCenter) (*entity.CostCenter, error)
	GetByCode(ctx context.Context, code int32) (*entity.CostCenter, error)
	List(ctx context.Context) ([]*entity.CostCenter, error)
	ListByType(ctx context.Context, ccType string) ([]*entity.CostCenter, error)
	Delete(ctx context.Context, code int32) error
}
