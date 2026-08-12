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

type CreateItemUseCase struct {
	Repo repository.ItemRepository
	Auth ports.AuthService
}

func NewCreateItemUseCase(
	repo repository.ItemRepository,
	auth ports.AuthService,
) *CreateItemUseCase {
	return &CreateItemUseCase{
		Repo: repo,
		Auth: auth,
	}
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
