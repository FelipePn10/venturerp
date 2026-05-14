package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
)

type ProductionPlanRepository interface {
	Create(ctx context.Context, plan *entity.ProductionPlan) (*entity.ProductionPlan, error)
	Update(ctx context.Context, plan *entity.ProductionPlan) (*entity.ProductionPlan, error)
	GetByCode(ctx context.Context, code int64) (*entity.ProductionPlan, error)
	List(ctx context.Context) ([]*entity.ProductionPlan, error)
	Delete(ctx context.Context, code int64) error
}
