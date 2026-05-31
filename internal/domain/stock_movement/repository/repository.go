package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
)

type StockMovementTypeRepository interface {
	Create(ctx context.Context, smt *entity.StockMovementType) (*entity.StockMovementType, error)
	Update(ctx context.Context, smt *entity.StockMovementType) (*entity.StockMovementType, error)
	GetByID(ctx context.Context, id int64) (*entity.StockMovementType, error)
	GetBySigla(ctx context.Context, sigla string) (*entity.StockMovementType, error)
	List(ctx context.Context, onlyActive bool) ([]*entity.StockMovementType, error)
}
