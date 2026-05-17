package item_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type FindItemByCode struct {
	Repo repository.ItemRepository
	Auth ports.AuthService
}

func NewFindItemByCode(
	repo repository.ItemRepository,
	auth ports.AuthService,
) *FindItemByCode {
	return &FindItemByCode{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *FindItemByCode) Execute(
	ctx context.Context,
	dto request.FindItemByCodeDTO,
) (*entity.Item, error) {
	if !uc.Auth.FindItemByCode(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	code, err := valueobject.NewItemCode(int64(dto.Code))
	if err != nil {
		return nil, err
	}

	item, err := uc.Repo.FindItemByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return item, nil
}
