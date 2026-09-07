package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type GetRestrictionsByItemUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
	// Items traduz o código de negócio do item para a chave interna, para a
	// consulta encontrar o que o configurador gravou.
	Items any
}

func (uc *GetRestrictionsByItemUseCase) Execute(ctx context.Context, itemCode int64) ([]*response.RestrictionResponse, error) {
	if !uc.Auth.CanListRestrictions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	interno, err := resolverCodigoDoItem(ctx, uc.Items, &itemCode)
	if err != nil {
		return nil, err
	}
	list, err := uc.Repo.GetByItemCode(ctx, *interno)
	if err != nil {
		return nil, err
	}
	return toRestrictionResponses(list), nil
}
