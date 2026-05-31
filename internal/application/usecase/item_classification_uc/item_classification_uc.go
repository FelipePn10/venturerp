package item_classification_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
)

type ItemClassificationUseCase struct {
	Repo repository.ItemClassificationRepository
}

func New(repo repository.ItemClassificationRepository) *ItemClassificationUseCase {
	return &ItemClassificationUseCase{Repo: repo}
}

// ─── Masks ────────────────────────────────────────────────────────────────────

func (uc *ItemClassificationUseCase) CreateMask(ctx context.Context, dto request.CreateClassificationMaskDTO) (*entity.ItemClassificationMask, error) {
	if dto.Mask == "" || dto.Description == "" {
		return nil, errors.New("mask and description are required")
	}
	m := &entity.ItemClassificationMask{
		Mask:        dto.Mask,
		Description: dto.Description,
		IsActive:    true,
	}
	return uc.Repo.CreateClassificationMask(ctx, m)
}

func (uc *ItemClassificationUseCase) UpdateMask(ctx context.Context, dto request.UpdateClassificationMaskDTO) (*entity.ItemClassificationMask, error) {
	m := &entity.ItemClassificationMask{
		ID:          dto.ID,
		Description: dto.Description,
		IsActive:    dto.IsActive,
	}
	return uc.Repo.UpdateClassificationMask(ctx, m)
}

func (uc *ItemClassificationUseCase) GetMaskByCode(ctx context.Context, code int64) (*entity.ItemClassificationMask, error) {
	return uc.Repo.GetClassificationMaskByCode(ctx, code)
}

func (uc *ItemClassificationUseCase) ListMasks(ctx context.Context, onlyActive bool) ([]*entity.ItemClassificationMask, error) {
	return uc.Repo.ListClassificationMasks(ctx, onlyActive)
}

// ─── Classifications ──────────────────────────────────────────────────────────

func (uc *ItemClassificationUseCase) CreateClassification(ctx context.Context, dto request.CreateItemClassificationDTO) (*entity.ItemClassification, error) {
	if dto.Code == "" || dto.Description == "" {
		return nil, errors.New("code and description are required")
	}
	mask, err := uc.Repo.GetClassificationMaskByCode(ctx, dto.MaskCode)
	if err != nil {
		return nil, errors.New("mask not found")
	}

	level := 1
	if dto.ParentCode != nil {
		parent, err := uc.Repo.GetItemClassificationByCode(ctx, *dto.ParentCode, dto.MaskCode)
		if err != nil {
			return nil, errors.New("parent classification not found")
		}
		level = parent.Level + 1
	}

	c := &entity.ItemClassification{
		Code:        dto.Code,
		MaskID:      mask.ID,
		Level:       level,
		Description: dto.Description,
		IsActive:    true,
	}

	if dto.ParentCode != nil {
		parent, err := uc.Repo.GetItemClassificationByCode(ctx, *dto.ParentCode, dto.MaskCode)
		if err == nil {
			c.ParentID = &parent.ID
		}
	}

	return uc.Repo.CreateItemClassification(ctx, c)
}

func (uc *ItemClassificationUseCase) UpdateClassification(ctx context.Context, dto request.UpdateItemClassificationDTO) (*entity.ItemClassification, error) {
	c := &entity.ItemClassification{
		ID:          dto.ID,
		Description: dto.Description,
		IsActive:    dto.IsActive,
	}
	return uc.Repo.UpdateItemClassification(ctx, c)
}

func (uc *ItemClassificationUseCase) GetByCode(ctx context.Context, code string, maskCode int64) (*entity.ItemClassification, error) {
	return uc.Repo.GetItemClassificationByCode(ctx, code, maskCode)
}

func (uc *ItemClassificationUseCase) ListByMask(ctx context.Context, maskID int64, onlyActive bool) ([]*entity.ItemClassification, error) {
	return uc.Repo.ListItemClassificationsByMask(ctx, maskID, onlyActive)
}

func (uc *ItemClassificationUseCase) ListChildren(ctx context.Context, parentID int64, onlyActive bool) ([]*entity.ItemClassification, error) {
	return uc.Repo.ListItemClassificationChildren(ctx, parentID, onlyActive)
}
