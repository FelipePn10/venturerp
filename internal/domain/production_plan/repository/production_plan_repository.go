package repository

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
)

var ErrAlreadyExists = domainerr.NewConflict("já existe um plano de produção com este código")
var ErrNotFound = domainerr.NewNotFound("plano de produção não encontrado")

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
