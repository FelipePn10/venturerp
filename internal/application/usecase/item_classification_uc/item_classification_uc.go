package item_classification_uc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
)

var maskPattern = regexp.MustCompile(`^9+(\.9+)*$`)

func validateMask(mask string) error {
	if !maskPattern.MatchString(mask) {
		return errors.New("mascara invalida: use grupos de 9 separados por ponto, por exemplo 99.99.99")
	}
	return nil
}

func validateClassificationCode(code, mask string, parent *entity.ItemClassification) (int, error) {
	parts, maskParts := strings.Split(code, "."), strings.Split(mask, ".")
	if len(parts) > len(maskParts) {
		return 0, errors.New("codigo excede os niveis da mascara")
	}
	for i, part := range parts {
		if len(part) != len(maskParts[i]) {
			return 0, errors.New("codigo nao corresponde a mascara")
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return 0, errors.New("codigo da classificacao deve ser numerico")
			}
		}
	}
	if parent == nil && len(parts) != 1 {
		return 0, errors.New("classificacao raiz deve conter somente o primeiro nivel")
	}
	if parent != nil {
		if len(parts) != parent.Level+1 || !strings.HasPrefix(code, parent.Code+".") {
			return 0, errors.New("codigo filho deve iniciar com o codigo completo do pai")
		}
	}
	return len(parts), nil
}

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
	dto.Mask = strings.TrimSpace(dto.Mask)
	if err := validateMask(dto.Mask); err != nil {
		return nil, err
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

	var parent *entity.ItemClassification
	if dto.ParentCode != nil {
		parent, err = uc.Repo.GetItemClassificationByCode(ctx, *dto.ParentCode, dto.MaskCode)
		if err != nil {
			return nil, errors.New("parent classification not found")
		}
	}
	dto.Code = strings.TrimSpace(dto.Code)
	level, err := validateClassificationCode(dto.Code, mask.Mask, parent)
	if err != nil {
		return nil, err
	}

	c := &entity.ItemClassification{
		Code:        dto.Code,
		MaskID:      mask.ID,
		Level:       level,
		Description: dto.Description,
		IsActive:    true,
	}

	if dto.ParentCode != nil {
		c.ParentID = &parent.ID
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
