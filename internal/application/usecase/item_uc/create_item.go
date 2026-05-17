package item_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
)

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
) (*entity.Item, error) {
	if !uc.Auth.CanCreateItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	created, err := uc.Repo.Create(ctx, item)
	if err != nil {
		return nil, err
	}

	return created, nil
}
