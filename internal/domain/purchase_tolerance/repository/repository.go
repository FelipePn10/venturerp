package repository

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_tolerance/entity"
	"github.com/shopspring/decimal"
)

type Repository interface {
	Save(context.Context, *entity.Tolerance) (*entity.Tolerance, error)
	List(context.Context, int64, *int64) ([]*entity.Tolerance, error)
	Delete(context.Context, int64, int64) error
	Resolve(context.Context, int64, *int64, string, string, decimal.Decimal) (*entity.Tolerance, error)
}
