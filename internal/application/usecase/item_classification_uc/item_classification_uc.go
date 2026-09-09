package item_classification_uc

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
)

var maskPattern = regexp.MustCompile(`^9+(\.9+)*$`)

func validateMask(mask string) error {
	if !maskPattern.MatchString(mask) {
		return errorsuc.NewValidationError("máscara inválida: use grupos de 9 separados por ponto, por exemplo 99.99.99")
	}
	return nil
}

func validateClassificationCode(code, mask string, parent *entity.ItemClassification) (int, error) {
	parts, maskParts := strings.Split(code, "."), strings.Split(mask, ".")
	if len(parts) > len(maskParts) {
		return 0, errorsuc.NewValidationError("código excede os níveis da máscara " + mask)
	}
	for i, part := range parts {
		if len(part) != len(maskParts[i]) {
			return 0, errorsuc.NewValidationError("código não corresponde à máscara " + mask)
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return 0, errorsuc.NewValidationError("código da classificação deve ser numérico")
			}
		}
	}
	if parent == nil && len(parts) != 1 {
		return 0, errorsuc.NewValidationError("classificação raiz deve conter somente o primeiro nível da máscara")
	}
	if parent != nil {
		if len(parts) != parent.Level+1 || !strings.HasPrefix(code, parent.Code+".") {
			return 0, errorsuc.NewValidationError("código filho deve iniciar com o código completo do pai (" + parent.Code + ")")
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

// resolveMask carrega a máscara pelo código informado pela tela e devolve 404 em
// PT-BR quando ela não existe no tenant autenticado.
func (uc *ItemClassificationUseCase) resolveMask(ctx context.Context, maskCode int64) (*entity.ItemClassificationMask, error) {
	if maskCode <= 0 {
		return nil, errorsuc.NewValidationError("informe a máscara da classificação")
	}
	mask, err := uc.Repo.GetClassificationMaskByCode(ctx, maskCode)
	if err != nil || mask == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("máscara de classificação %d não encontrada nesta empresa", maskCode))
	}
	return mask, nil
}

// maskByID resolve a máscara a partir do id interno usando o catálogo do tenant,
// evitando uma consulta dedicada só para preencher o código na resposta.
func (uc *ItemClassificationUseCase) maskByID(ctx context.Context, maskID int64) *entity.ItemClassificationMask {
	masks, err := uc.Repo.ListClassificationMasks(ctx, false)
	if err != nil {
		return nil
	}
	for _, m := range masks {
		if m.ID == maskID {
			return m
		}
	}
	return nil
}

// viewFor monta o contexto de apresentação (máscara + irmãos) de uma máscara.
func (uc *ItemClassificationUseCase) viewFor(ctx context.Context, mask *entity.ItemClassificationMask) *classificationView {
	all, err := uc.Repo.ListItemClassificationsByMask(ctx, mask.ID, false)
	if err != nil {
		return newClassificationView(mask, nil)
	}
	return newClassificationView(mask, all)
}

// ─── Masks ────────────────────────────────────────────────────────────────────

func (uc *ItemClassificationUseCase) CreateMask(ctx context.Context, dto request.CreateClassificationMaskDTO) (*response.ItemClassificationMaskResponse, error) {
	dto.Mask, dto.Description = strings.TrimSpace(dto.Mask), strings.TrimSpace(dto.Description)
	if dto.Mask == "" {
		return nil, errorsuc.NewValidationError("informe a máscara (por exemplo 99.99.99)")
	}
	if dto.Description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição da máscara")
	}
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
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("informe a descrição da máscara")
	}
	id := dto.ID
	// A tela identifica a máscara pelo código de negócio, não pelo id interno.
	if id == 0 {
		if dto.Code == 0 {
			return nil, errorsuc.NewValidationError("informe o código da máscara a alterar")
		}
		mask, err := uc.resolveMask(ctx, dto.Code)
		if err != nil {
			return nil, err
		}
		id = mask.ID
	}
	m := &entity.ItemClassificationMask{
		ID:          id,
		Description: strings.TrimSpace(dto.Description),
		IsActive:    ptrutil.BoolOrTrue(dto.IsActive),
	}
	updated, err := uc.Repo.UpdateClassificationMask(ctx, m)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, errorsuc.NewNotFoundError("máscara de classificação não encontrada nesta empresa")
	}
	return toClassificationMaskResponse(updated), nil
}

func (uc *ItemClassificationUseCase) ListMasks(ctx context.Context, onlyActive bool) ([]*response.ItemClassificationMaskResponse, error) {
	list, err := uc.Repo.ListClassificationMasks(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toClassificationMaskResponses(list), nil
}

func (uc *ItemClassificationUseCase) GetMaskByCode(ctx context.Context, code int64) (*response.ItemClassificationMaskResponse, error) {
	m, err := uc.resolveMask(ctx, code)
	if err != nil {
		return nil, err
	}
	return toClassificationMaskResponse(m), nil
}

// ─── Classifications ──────────────────────────────────────────────────────────

// normalizeParentCode trata "" / "   " como ausência de pai — a tela envia string
// vazia ao cadastrar uma classificação raiz.
func normalizeParentCode(parent *string) *string {
	if parent == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*parent)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (uc *ItemClassificationUseCase) CreateClassification(ctx context.Context, dto request.CreateItemClassificationDTO) (*response.ItemClassificationResponse, error) {
	dto.Code, dto.Description = strings.TrimSpace(dto.Code), strings.TrimSpace(dto.Description)
	if dto.Code == "" {
		return nil, errorsuc.NewValidationError("informe o código da classificação")
	}
	if dto.Description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição da classificação")
	}
	mask, err := uc.resolveMask(ctx, dto.MaskCode)
	if err != nil {
		return nil, err
	}

	var parent *entity.ItemClassification
	parentCode := normalizeParentCode(dto.ParentCode)
	if parentCode != nil {
		parent, err = uc.Repo.GetItemClassificationByCode(ctx, *parentCode, dto.MaskCode)
		if err != nil || parent == nil {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("classificação pai %s não encontrada na máscara %d", *parentCode, dto.MaskCode))
		}
	}
	level, err := validateClassificationCode(dto.Code, mask.Mask, parent)
	if err != nil {
		return nil, err
	}
	if existing, getErr := uc.Repo.GetItemClassificationByCode(ctx, dto.Code, dto.MaskCode); getErr == nil && existing != nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("a classificação %s já existe nesta máscara", dto.Code))
	}

	c := &entity.ItemClassification{
		Code:        dto.Code,
		MaskID:      mask.ID,
		Level:       level,
		Description: dto.Description,
		IsActive:    true,
	}
	if parent != nil {
		c.ParentID = &parent.ID
	}

	created, err := uc.Repo.CreateItemClassification(ctx, c)
	if err != nil {
		return nil, err
	}
	return uc.viewFor(ctx, mask).toResponse(created), nil
}

func (uc *ItemClassificationUseCase) UpdateClassification(ctx context.Context, dto request.UpdateItemClassificationDTO) (*response.ItemClassificationResponse, error) {
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("informe a descrição da classificação")
	}
	var mask *entity.ItemClassificationMask
	id := dto.ID
	// A tela identifica a classificação por código + máscara.
	if id == 0 {
		code := strings.TrimSpace(dto.Code)
		if code == "" {
			return nil, errorsuc.NewValidationError("informe o código da classificação a alterar")
		}
		var err error
		if mask, err = uc.resolveMask(ctx, dto.MaskCode); err != nil {
			return nil, err
		}
		current, err := uc.Repo.GetItemClassificationByCode(ctx, code, dto.MaskCode)
		if err != nil || current == nil {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("classificação %s não encontrada na máscara %d", code, dto.MaskCode))
		}
		id = current.ID
	}
	c := &entity.ItemClassification{
		ID:          id,
		Description: strings.TrimSpace(dto.Description),
		IsActive:    ptrutil.BoolOrTrue(dto.IsActive),
	}
	updated, err := uc.Repo.UpdateItemClassification(ctx, c)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, errorsuc.NewNotFoundError("classificação não encontrada nesta empresa")
	}
	if mask == nil {
		mask, _ = uc.Repo.GetClassificationMaskByCode(ctx, dto.MaskCode)
	}
	if mask == nil {
		return newClassificationView(nil, nil).toResponse(updated), nil
	}
	return uc.viewFor(ctx, mask).toResponse(updated), nil
}

func (uc *ItemClassificationUseCase) GetByCode(ctx context.Context, code string, maskCode int64) (*response.ItemClassificationResponse, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errorsuc.NewValidationError("informe o código da classificação")
	}
	mask, err := uc.resolveMask(ctx, maskCode)
	if err != nil {
		return nil, err
	}
	c, err := uc.Repo.GetItemClassificationByCode(ctx, code, maskCode)
	if err != nil || c == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("classificação %s não encontrada na máscara %d", code, maskCode))
	}
	return uc.viewFor(ctx, mask).toResponse(c), nil
}

// ListByMaskCode lista as classificações de uma máscara identificada pelo seu
// código de negócio (o que a tela conhece), e não pelo id interno.
func (uc *ItemClassificationUseCase) ListByMaskCode(ctx context.Context, maskCode int64, onlyActive bool) ([]*response.ItemClassificationResponse, error) {
	mask, err := uc.resolveMask(ctx, maskCode)
	if err != nil {
		return nil, err
	}
	list, err := uc.Repo.ListItemClassificationsByMask(ctx, mask.ID, onlyActive)
	if err != nil {
		return nil, err
	}
	return uc.viewFor(ctx, mask).toResponses(list), nil
}

func (uc *ItemClassificationUseCase) ListChildren(ctx context.Context, parentID int64, onlyActive bool) ([]*response.ItemClassificationResponse, error) {
	if parentID <= 0 {
		return nil, errorsuc.NewValidationError("informe a classificação pai")
	}
	list, err := uc.Repo.ListItemClassificationChildren(ctx, parentID, onlyActive)
	if err != nil {
		return nil, errorsuc.NewNotFoundError("classificação pai não encontrada nesta empresa")
	}
	if len(list) == 0 {
		return []*response.ItemClassificationResponse{}, nil
	}
	mask := uc.maskByID(ctx, list[0].MaskID)
	if mask == nil {
		return newClassificationView(nil, nil).toResponses(list), nil
	}
	return uc.viewFor(ctx, mask).toResponses(list), nil
}

func (uc *ItemClassificationUseCase) ListCatalog(ctx context.Context, onlyActive bool) ([]response.ItemClassificationCatalogResponse, error) {
	masks, err := uc.Repo.ListClassificationMasks(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	result := make([]response.ItemClassificationCatalogResponse, 0)
	for _, mask := range masks {
		classifications, listErr := uc.Repo.ListItemClassificationsByMask(ctx, mask.ID, onlyActive)
		if listErr != nil {
			return nil, listErr
		}
		for _, classification := range classifications {
			result = append(result, response.ItemClassificationCatalogResponse{
				ID: classification.ID, Code: classification.Code, MaskID: mask.ID,
				MaskCode: mask.Code, Mask: mask.Mask, Description: classification.Description,
				MaskDescription: mask.Description, IsActive: classification.IsActive,
			})
		}
	}
	return result, nil
}
