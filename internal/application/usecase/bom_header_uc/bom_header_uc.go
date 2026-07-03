package bom_header_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository"
)

type BomHeaderUseCase struct {
	repo repository.BomHeaderRepository
}

func New(repo repository.BomHeaderRepository) *BomHeaderUseCase {
	return &BomHeaderUseCase{repo: repo}
}

// Create opens a new BOM header for an item, auto-assigning the next version.
func (uc *BomHeaderUseCase) Create(ctx context.Context, dto request.CreateBomHeaderDTO) (*response.BomHeaderResponse, error) {
	mask := ""
	if dto.Mask != nil {
		mask = *dto.Mask
	}
	version, err := uc.repo.NextVersion(ctx, dto.ItemCode, mask)
	if err != nil {
		return nil, fmt.Errorf("computing next version: %w", err)
	}
	h, err := entity.NewBomHeader(dto.ItemCode, dto.Mask, dto.BomType, version, dto.ValidFrom, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, h)
	if err != nil {
		return nil, err
	}
	return toResponse(created), nil
}

func (uc *BomHeaderUseCase) Get(ctx context.Context, id int64) (*response.BomHeaderResponse, error) {
	h, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bom header not found: %w", err)
	}
	return toResponse(h), nil
}

func (uc *BomHeaderUseCase) ListByItem(ctx context.Context, itemCode int64) ([]*response.BomHeaderResponse, error) {
	hs, err := uc.repo.ListByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.BomHeaderResponse, 0, len(hs))
	for _, h := range hs {
		out = append(out, toResponse(h))
	}
	return out, nil
}

// UpdateStatus moves the BOM through its approval lifecycle (DRAFT→APPROVED→OBSOLETE).
func (uc *BomHeaderUseCase) UpdateStatus(ctx context.Context, dto request.UpdateBomHeaderStatusDTO) (*response.BomHeaderResponse, error) {
	if !entity.ValidStatus(dto.Status) {
		return nil, fmt.Errorf("invalid status %q (expected DRAFT, APPROVED or OBSOLETE)", dto.Status)
	}
	updated, err := uc.repo.UpdateStatus(ctx, dto.ID, dto.Status)
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

func toResponse(h *entity.BomHeader) *response.BomHeaderResponse {
	return &response.BomHeaderResponse{
		ID:        h.ID,
		ItemCode:  h.ItemCode,
		Mask:      h.Mask,
		BomType:   h.BomType,
		Version:   h.Version,
		Status:    h.Status,
		ValidFrom: h.ValidFrom,
		IsActive:  h.IsActive,
		CreatedAt: h.CreatedAt,
	}
}
