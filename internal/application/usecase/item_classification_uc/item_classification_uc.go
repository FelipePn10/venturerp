package item_classification_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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

func (uc *ItemClassificationUseCase) CreateMask(ctx context.Context, dto request.CreateClassificationMaskDTO) (*response.ItemClassificationMaskResponse, error) {
	if dto.Mask == "" || dto.Description == "" {
		return nil, errors.New("mask and description are required")
	}
	m := &entity.ItemClassificationMask{
		Mask:        dto.Mask,
		Description: dto.Description,
		IsActive:    true,
	}
	created, err := uc.Repo.CreateClassificationMask(ctx, m)
	if err != nil {
		return nil, err
	}
	return toClassificationMaskResponse(created), nil
}

func (uc *ItemClassificationUseCase) UpdateMask(ctx context.Context, dto request.UpdateClassificationMaskDTO) (*response.ItemClassificationMaskResponse, error) {
	m := &entity.ItemClassificationMask{
		ID:          dto.ID,
		Description: dto.Description,
		IsActive:    dto.IsActive,
	}
	updated, err := uc.Repo.UpdateClassificationMask(ctx, m)
	if err != nil {
		return nil, err
	}
	return toClassificationMaskResponse(updated), nil
}

func (uc *ItemClassificationUseCase) GetMaskByCode(ctx context.Context, code int64) (*response.ItemClassificationMaskResponse, error) {
	m, err := uc.Repo.GetClassificationMaskByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toClassificationMaskResponse(m), nil
}

func (uc *ItemClassificationUseCase) ListMasks(ctx context.Context, onlyActive bool) ([]*response.ItemClassificationMaskResponse, error) {
	list, err := uc.Repo.ListClassificationMasks(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toClassificationMaskResponses(list), nil
}

// ─── Classifications ──────────────────────────────────────────────────────────

func (uc *ItemClassificationUseCase) CreateClassification(ctx context.Context, dto request.CreateItemClassificationDTO) (*response.ItemClassificationResponse, error) {
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

	created, err := uc.Repo.CreateItemClassification(ctx, c)
	if err != nil {
		return nil, err
	}
	return toItemClassificationResponse(created), nil
}

func (uc *ItemClassificationUseCase) UpdateClassification(ctx context.Context, dto request.UpdateItemClassificationDTO) (*response.ItemClassificationResponse, error) {
	c := &entity.ItemClassification{
		ID:          dto.ID,
		Description: dto.Description,
		IsActive:    dto.IsActive,
	}
	updated, err := uc.Repo.UpdateItemClassification(ctx, c)
	if err != nil {
		return nil, err
	}
	return toItemClassificationResponse(updated), nil
}

func (uc *ItemClassificationUseCase) GetByCode(ctx context.Context, code string, maskCode int64) (*response.ItemClassificationResponse, error) {
	c, err := uc.Repo.GetItemClassificationByCode(ctx, code, maskCode)
	if err != nil {
		return nil, err
	}
	return toItemClassificationResponse(c), nil
}

func (uc *ItemClassificationUseCase) ListByMask(ctx context.Context, maskID int64, onlyActive bool) ([]*response.ItemClassificationResponse, error) {
	list, err := uc.Repo.ListItemClassificationsByMask(ctx, maskID, onlyActive)
	if err != nil {
		return nil, err
	}
	return toItemClassificationResponses(list), nil
}

func (uc *ItemClassificationUseCase) ListChildren(ctx context.Context, parentID int64, onlyActive bool) ([]*response.ItemClassificationResponse, error) {
	list, err := uc.Repo.ListItemClassificationChildren(ctx, parentID, onlyActive)
	if err != nil {
		return nil, err
	}
	return toItemClassificationResponses(list), nil
}
