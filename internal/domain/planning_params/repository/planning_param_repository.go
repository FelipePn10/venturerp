package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
	"github.com/google/uuid"
)

type PlanningParamRepository interface {
	GetByNumber(ctx context.Context, paramNumber int) (*entity.PlanningParam, error)
	GetByKey(ctx context.Context, key string) (*entity.PlanningParam, error)
	List(ctx context.Context) ([]*entity.PlanningParam, error)
	Update(ctx context.Context, paramNumber int, value string, updatedBy uuid.UUID) (*entity.PlanningParam, error)
}
