package item_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

var ErrItemBaseNotFound = errors.New("O item-base informado não existe.")

type automaticBusinessCodeRepository interface {
	NextAutomaticBusinessCode(context.Context, int64) (valueobject.BusinessCode, error)
}

type pdmReferenceRepository interface {
	ValidatePDMReferences(context.Context, int64, int, int) error
}

type CreateItemUseCase struct {
	Repo repository.ItemRepository
	Auth ports.AuthService
	// FiscalCatalog valida as classificações fiscais contra o cadastro canônico
	// (/api/fiscal-classifications). Opcional: sem ele os códigos passam livres.
	FiscalCatalog fiscalClassificationCatalog
}

func NewCreateItemUseCase(
	repo repository.ItemRepository,
	auth ports.AuthService,
	fiscalCatalog ...fiscalClassificationCatalog,
) *CreateItemUseCase {
	uc := &CreateItemUseCase{
		Repo: repo,
		Auth: auth,
	}
	if len(fiscalCatalog) > 0 {
		uc.FiscalCatalog = fiscalCatalog[0]
	}
	return uc
}

func (uc *CreateItemUseCase) Execute(
	ctx context.Context,
	item *entity.Item,
) (*response.ItemResponse, error) {
	if !uc.Auth.CanCreateItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	item.EnterpriseID = enterpriseID
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	item.CreatedBy = userID
	if item.BusinessCode == "" {
		generator, ok := uc.Repo.(automaticBusinessCodeRepository)
		if !ok {
			return nil, fmt.Errorf("geracao automatica de codigo indisponivel")
		}
		item.BusinessCode, err = generator.NextAutomaticBusinessCode(ctx, enterpriseID)
		if err != nil {
			return nil, err
		}
	}
	if !item.BusinessCode.IsValid() {
		return nil, entity.ErrInvalidCode
	}
	if item.PDM.GroupCode != 0 || item.PDM.ModifierCode != 0 {
		validator, ok := uc.Repo.(pdmReferenceRepository)
		if !ok {
			return nil, fmt.Errorf("validação de referências PDM indisponível")
		}
		if err = validator.ValidatePDMReferences(ctx, enterpriseID, int(item.PDM.GroupCode), int(item.PDM.ModifierCode)); err != nil {
			return nil, err
		}
	}
	// Item-base e item de embalagem chegam como código de negócio (texto).
	if err = resolveReferenceCodes(ctx, uc.Repo, item); err != nil {
		return nil, err
	}
	if err = validateFiscalClassifications(ctx, uc.FiscalCatalog, enterpriseID, item); err != nil {
		return nil, err
	}
	if item.Engineering.ItemBaseCod != nil {
		code, err := valueobject.NewItemCode(int64(*item.Engineering.ItemBaseCod))
		if err != nil {
			return nil, ErrItemBaseNotFound
		}
		if _, err = uc.Repo.FindItemByCode(ctx, code); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrItemBaseNotFound
			}
			return nil, fmt.Errorf("validar item-base: %w", err)
		}
	}

	created, err := uc.Repo.Create(ctx, item)
	if err != nil {
		return nil, err
	}

	return toItemResponse(created), nil
}
