package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
)

type BomHeaderRepository interface {
	Create(ctx context.Context, h *entity.BomHeader) (*entity.BomHeader, error)
	GetByID(ctx context.Context, id int64) (*entity.BomHeader, error)
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.BomHeader, error)
	UpdateStatus(ctx context.Context, id int64, status string) (*entity.BomHeader, error)
	NextVersion(ctx context.Context, itemCode int64, mask string) (int32, error)
}
