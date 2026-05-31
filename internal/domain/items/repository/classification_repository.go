package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

type ItemClassificationRepository interface {
	// ── Masks ──────────────────────────────────────────────────────────────
	CreateClassificationMask(ctx context.Context, m *entity.ItemClassificationMask) (*entity.ItemClassificationMask, error)
	UpdateClassificationMask(ctx context.Context, m *entity.ItemClassificationMask) (*entity.ItemClassificationMask, error)
	GetClassificationMaskByCode(ctx context.Context, code int64) (*entity.ItemClassificationMask, error)
	ListClassificationMasks(ctx context.Context, onlyActive bool) ([]*entity.ItemClassificationMask, error)
	NextClassificationMaskCode(ctx context.Context) (int64, error)

	// ── Classifications ────────────────────────────────────────────────────
	CreateItemClassification(ctx context.Context, c *entity.ItemClassification) (*entity.ItemClassification, error)
	UpdateItemClassification(ctx context.Context, c *entity.ItemClassification) (*entity.ItemClassification, error)
	GetItemClassificationByCode(ctx context.Context, code string, maskCode int64) (*entity.ItemClassification, error)
	ListItemClassificationsByMask(ctx context.Context, maskID int64, onlyActive bool) ([]*entity.ItemClassification, error)
	ListItemClassificationChildren(ctx context.Context, parentID int64, onlyActive bool) ([]*entity.ItemClassification, error)
}
