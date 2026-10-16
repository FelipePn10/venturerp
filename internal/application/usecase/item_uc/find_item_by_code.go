package item_uc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	// Descricao deixa o mesmo campo aceitar o código OU a descrição do item.
	// Opcional: sem ele, só o código vale.
	Descricao BuscaItemPorDescricao
}

// ItemPorDescricao é um item que bate com a descrição procurada.
type ItemPorDescricao struct {
	Codigo    string
	Descricao string
}

// BuscaItemPorDescricao procura itens da empresa pela descrição.
type BuscaItemPorDescricao interface {
	ItensPorDescricao(ctx context.Context, texto string, limite int) ([]ItemPorDescricao, error)
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
		// Não tem formato de código (tem espaço, aspas…): pode ser a descrição.
		if porDescricao, ok, errDesc := uc.porDescricao(ctx, dto.Code); errDesc != nil || ok {
			return porDescricao, errDesc
		}
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
						if err := fillReferenceBusinessCodes(ctx, uc.Repo, item); err != nil {
							return nil, err
						}
						return toItemResponse(item), nil
					}
				}
			}
			if porDescricao, ok, errDesc := uc.porDescricao(ctx, dto.Code); errDesc != nil || ok {
				return porDescricao, errDesc
			}
			return nil, errorsuc.ErrProductNotFound
		}
		return nil, err
	}

	// A tela trabalha por código de negócio: traduz as referências antes de
	// devolver, para que um cadastro parcial reabra exatamente como foi salvo.
	if err := fillReferenceBusinessCodes(ctx, uc.Repo, item); err != nil {
		return nil, err
	}
	return toItemResponse(item), nil
}

// porDescricao tenta o texto como descrição. Um item só: devolve-o. Vários: 422
// listando os códigos para a pessoa escolher. Nenhum: ok=false (segue o fluxo
// do código, que diz "não encontrado").
func (uc *FindItemByCode) porDescricao(ctx context.Context, texto string) (*response.ItemResponse, bool, error) {
	if uc.Descricao == nil || len(strings.TrimSpace(texto)) < 3 {
		return nil, false, nil
	}
	achados, err := uc.Descricao.ItensPorDescricao(ctx, texto, 6)
	if err != nil {
		return nil, true, err
	}
	switch len(achados) {
	case 0:
		return nil, false, nil
	case 1:
		code, err := valueobject.NewBusinessCode(achados[0].Codigo)
		if err != nil {
			return nil, true, err
		}
		item, err := uc.Repo.FindItemByBusinessCode(ctx, code)
		if err != nil {
			return nil, true, err
		}
		if err := fillReferenceBusinessCodes(ctx, uc.Repo, item); err != nil {
			return nil, true, err
		}
		return toItemResponse(item), true, nil
	}
	nomes := make([]string, 0, len(achados))
	for i, a := range achados {
		if i == 5 {
			nomes = append(nomes, "…")
			break
		}
		nomes = append(nomes, fmt.Sprintf("%s (%s)", a.Codigo, strings.TrimSpace(a.Descricao)))
	}
	return nil, true, errorsuc.NewValidationError(fmt.Sprintf(
		"mais de um item tem essa descrição: %s — digite o código do item", strings.Join(nomes, "; ")))
}
