package item_classification

import (
	"context"
	"fmt"

	itemEntity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type ItemClassificationRepositorySQLC struct {
	q *sqlc.Queries
}

var _ domainrepo.ItemClassificationRepository = (*ItemClassificationRepositorySQLC)(nil)

func New(q *sqlc.Queries) *ItemClassificationRepositorySQLC {
	return &ItemClassificationRepositorySQLC{q: q}
}

// ─── Masks ────────────────────────────────────────────────────────────────────

func (r *ItemClassificationRepositorySQLC) CreateClassificationMask(ctx context.Context, m *itemEntity.ItemClassificationMask) (*itemEntity.ItemClassificationMask, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateClassificationMask(ctx, sqlc.CreateClassificationMaskParams{
		EnterpriseID: e,
		Mask:         m.Mask,
		Description:  m.Description,
	})
	if err != nil {
		return nil, err
	}
	return maskToEntity(row), nil
}

func (r *ItemClassificationRepositorySQLC) UpdateClassificationMask(ctx context.Context, m *itemEntity.ItemClassificationMask) (*itemEntity.ItemClassificationMask, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateClassificationMask(ctx, sqlc.UpdateClassificationMaskParams{
		ID:           m.ID,
		Description:  m.Description,
		IsActive:     m.IsActive,
		EnterpriseID: e,
	})
	if err != nil {
		return nil, err
	}
	return maskToEntity(row), nil
}

func (r *ItemClassificationRepositorySQLC) GetClassificationMaskByCode(ctx context.Context, code int64) (*itemEntity.ItemClassificationMask, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetClassificationMaskByCode(ctx, sqlc.GetClassificationMaskByCodeParams{Code: code, EnterpriseID: e})
	if err != nil {
		return nil, err
	}
	return maskToEntity(row), nil
}

func (r *ItemClassificationRepositorySQLC) ListClassificationMasks(ctx context.Context, onlyActive bool) ([]*itemEntity.ItemClassificationMask, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListClassificationMasks(ctx, sqlc.ListClassificationMasksParams{EnterpriseID: e, Column2: onlyActive})
	if err != nil {
		return nil, err
	}
	result := make([]*itemEntity.ItemClassificationMask, len(rows))
	for i, row := range rows {
		result[i] = maskToEntity(row)
	}
	return result, nil
}

func (r *ItemClassificationRepositorySQLC) NextClassificationMaskCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextClassificationMaskCode(ctx)
	if err != nil {
		return 0, err
	}
	return int64(code), nil
}

func maskToEntity(row sqlc.ItemClassificationMask) *itemEntity.ItemClassificationMask {
	return &itemEntity.ItemClassificationMask{
		ID:          row.ID,
		Code:        row.Code,
		Mask:        row.Mask,
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Classifications ──────────────────────────────────────────────────────────

func (r *ItemClassificationRepositorySQLC) CreateItemClassification(ctx context.Context, c *itemEntity.ItemClassification) (*itemEntity.ItemClassification, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ok, err := r.q.ClassificationMaskBelongsToEnterprise(ctx, sqlc.ClassificationMaskBelongsToEnterpriseParams{ID: c.MaskID, EnterpriseID: e})
	if err != nil || !ok {
		return nil, fmt.Errorf("mascara nao pertence a empresa")
	}
	row, err := r.q.CreateItemClassification(ctx, sqlc.CreateItemClassificationParams{
		Code:        c.Code,
		MaskID:      c.MaskID,
		ParentID:    c.ParentID,
		Level:       int32(c.Level),
		Description: c.Description,
	})
	if err != nil {
		return nil, err
	}
	return classificationToEntity(row), nil
}

func (r *ItemClassificationRepositorySQLC) UpdateItemClassification(ctx context.Context, c *itemEntity.ItemClassification) (*itemEntity.ItemClassification, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ok, err := r.q.ItemClassificationBelongsToEnterprise(ctx, sqlc.ItemClassificationBelongsToEnterpriseParams{ID: c.ID, EnterpriseID: e})
	if err != nil || !ok {
		return nil, fmt.Errorf("classificacao nao pertence a empresa")
	}
	row, err := r.q.UpdateItemClassification(ctx, sqlc.UpdateItemClassificationParams{
		ID:          c.ID,
		Description: c.Description,
		IsActive:    c.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return classificationToEntity(row), nil
}

func (r *ItemClassificationRepositorySQLC) GetItemClassificationByCode(ctx context.Context, code string, maskCode int64) (*itemEntity.ItemClassification, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetItemClassificationByCode(ctx, sqlc.GetItemClassificationByCodeParams{
		Code:         code,
		Code_2:       maskCode,
		EnterpriseID: e,
	})
	if err != nil {
		return nil, err
	}
	return classificationToEntity(row), nil
}

func (r *ItemClassificationRepositorySQLC) ListItemClassificationsByMask(ctx context.Context, maskID int64, onlyActive bool) ([]*itemEntity.ItemClassification, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ok, err := r.q.ClassificationMaskBelongsToEnterprise(ctx, sqlc.ClassificationMaskBelongsToEnterpriseParams{ID: maskID, EnterpriseID: e})
	if err != nil || !ok {
		return nil, fmt.Errorf("mascara nao pertence a empresa")
	}
	rows, err := r.q.ListItemClassificationsByMask(ctx, sqlc.ListItemClassificationsByMaskParams{
		MaskID:  maskID,
		Column2: onlyActive,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*itemEntity.ItemClassification, len(rows))
	for i, row := range rows {
		result[i] = classificationToEntity(row)
	}
	return result, nil
}

func (r *ItemClassificationRepositorySQLC) ListItemClassificationChildren(ctx context.Context, parentID int64, onlyActive bool) ([]*itemEntity.ItemClassification, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ok, err := r.q.ItemClassificationBelongsToEnterprise(ctx, sqlc.ItemClassificationBelongsToEnterpriseParams{ID: parentID, EnterpriseID: e})
	if err != nil || !ok {
		return nil, fmt.Errorf("classificacao pai nao pertence a empresa")
	}
	rows, err := r.q.ListItemClassificationChildren(ctx, sqlc.ListItemClassificationChildrenParams{
		ParentID: &parentID,
		Column2:  onlyActive,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*itemEntity.ItemClassification, len(rows))
	for i, row := range rows {
		result[i] = classificationToEntity(row)
	}
	return result, nil
}

func classificationToEntity(row sqlc.ItemClassification) *itemEntity.ItemClassification {
	return &itemEntity.ItemClassification{
		ID:          row.ID,
		Code:        row.Code,
		MaskID:      row.MaskID,
		ParentID:    row.ParentID,
		Level:       int(row.Level),
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}
