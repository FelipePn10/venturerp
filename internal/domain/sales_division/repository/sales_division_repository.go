package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
)

type SalesDivisionRepository interface {
	Create(ctx context.Context, sd *entity.SalesDivision) (*entity.SalesDivision, error)
	Update(ctx context.Context, sd *entity.SalesDivision) (*entity.SalesDivision, error)
	GetByCode(ctx context.Context, code int64) (*entity.SalesDivision, error)
	List(ctx context.Context) ([]*entity.SalesDivision, error)
	ListActive(ctx context.Context) ([]*entity.SalesDivision, error)
	Delete(ctx context.Context, code int64) error
}
