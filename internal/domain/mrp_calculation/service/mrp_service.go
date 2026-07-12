package service

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type MRPService interface {
	Calculate(ctx context.Context, planCode, initialOrderNumber int64, generateLLC bool) (*entity.MRPCalculationLog, error)
	GenerateLLC(ctx context.Context) error
	CalculateItemLLC(ctx context.Context, itemCode int64) (int, error)
	CalculateNetRequirements(ctx context.Context, input *entity.MRPInput) (*entity.MRPOutput, error)
	ExplodeStructure(ctx context.Context, parentCode int64, mask string, quantity float64, level int) ([]*entity.MRPInput, error)
}
