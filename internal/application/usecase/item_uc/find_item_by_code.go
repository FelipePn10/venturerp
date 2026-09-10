package item_uc

import (
	"context"
	"errors"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type itemByBusinessCodeRepository interface {
	FindItemByBusinessCode(context.Context, valueobject.BusinessCode) (*entity.Item, error)
}

type FindItemByCode struct {
	Repo itemByBusinessCodeRepository
	Auth ports.AuthService
}

func NewFindItemByCode(
	repo itemByBusinessCodeRepository,
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
) (*response.ItemResponse, error) {
	if !uc.Auth.FindItemByCode(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	code, err := valueobject.NewBusinessCode(dto.Code)
	if err != nil {
		return nil, err
	}

	item, err := uc.Repo.FindItemByBusinessCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// O middleware de compatibilidade pode entregar aqui a chave legada
			// numérica depois de resolver um código comercial da URL. Aceitar esse
			// formato evita que uma busca válida por "TP-01001-A" vire um falso 404.
			legacy, parseErr := strconv.ParseInt(string(code), 10, 64)
			if parseErr == nil && legacy > 0 {
				if finder, ok := any(uc.Repo).(interface {
					FindItemByCode(context.Context, valueobject.ItemCode) (*entity.Item, error)
				}); ok {
					item, err = finder.FindItemByCode(ctx, valueobject.ItemCode(legacy))
					if err == nil {
						fillReferenceBusinessCodes(ctx, uc.Repo, item)
						return toItemResponse(item), nil
					}
				}
			}
			return nil, errorsuc.ErrProductNotFound
		}
		return nil, err
	}

	// A tela trabalha por código de negócio: traduz as referências antes de
	// devolver, para que um cadastro parcial reabra exatamente como foi salvo.
	fillReferenceBusinessCodes(ctx, uc.Repo, item)
	return toItemResponse(item), nil
}
