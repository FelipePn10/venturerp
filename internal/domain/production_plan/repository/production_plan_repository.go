package repository

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
)

var ErrAlreadyExists = errors.New("production plan code already exists")

type ProductionPlanRepository interface {
	Create(ctx context.Context, plan *entity.ProductionPlan) (*entity.ProductionPlan, error)
	Update(ctx context.Context, plan *entity.ProductionPlan) (*entity.ProductionPlan, error)
	GetByCode(ctx context.Context, code int64) (*entity.ProductionPlan, error)
	List(ctx context.Context) ([]*entity.ProductionPlan, error)
	Delete(ctx context.Context, code int64) error
	UpdateLastCalculated(ctx context.Context, code int64) error
	ReplaceInterFactories(ctx context.Context, planCode int64, entries []*entity.InterFactoryEnterprise) ([]*entity.InterFactoryEnterprise, error)
	ListInterFactories(ctx context.Context, planCode int64) ([]*entity.InterFactoryEnterprise, error)
}
