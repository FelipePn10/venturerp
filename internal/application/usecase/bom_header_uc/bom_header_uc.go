package bom_header_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type BomHeaderUseCase struct {
	repo repository.BomHeaderRepository
	// items resolve o código de negócio do item (texto) para a chave legada.
	items any
	auth  ports.AuthService
}

func New(repo repository.BomHeaderRepository, items any, auth ports.AuthService) *BomHeaderUseCase {
	return &BomHeaderUseCase{repo: repo, items: items, auth: auth}
}

// resolveItem devolve o item da empresa autenticada a partir do código de texto.
func (uc *BomHeaderUseCase) resolveItem(ctx context.Context, code request.TextCode) (*itementity.Item, error) {
	return itemresolution.Resolve(ctx, uc.items, code)
}

// normalizeBomType aceita o tipo em qualquer caixa e cai no padrão MBOM.
func normalizeBomType(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	switch value {
	case "":
		return "MBOM", nil
	case "MBOM", "EBOM":
		return value, nil
	default:
		return "", errorsuc.NewValidationError("tipo de estrutura inválido: use MBOM (fabricação) ou EBOM (engenharia)")
	}
}

// Create abre um novo cabeçalho de estrutura para o item, atribuindo a próxima
// versão automaticamente.
func (uc *BomHeaderUseCase) Create(ctx context.Context, dto request.CreateBomHeaderDTO) (*response.BomHeaderResponse, error) {
	item, err := uc.resolveItem(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	bomType, err := normalizeBomType(dto.BomType)
	if err != nil {
		return nil, err
	}
	// O autor é sempre o usuário autenticado.
	actor, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	mask := ""
	if dto.Mask != nil {
		mask = strings.TrimSpace(*dto.Mask)
	}
	var maskPtr *string
	if mask != "" {
		maskPtr = &mask
	}
	version, err := uc.repo.NextVersion(ctx, int64(item.Code), mask)
	if err != nil {
		return nil, err
	}
	h, err := entity.NewBomHeader(int64(item.Code), maskPtr, bomType, version, dto.ValidFrom, actor)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, h)
	if err != nil {
		return nil, err
	}
	return toResponse(created, item), nil
}

func (uc *BomHeaderUseCase) Get(ctx context.Context, id int64) (*response.BomHeaderResponse, error) {
	h, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(h, uc.itemOf(ctx, h.ItemCode)), nil
}

// ListByItem lista as versões do cabeçalho pelo código de negócio do item.
func (uc *BomHeaderUseCase) ListByItem(ctx context.Context, code request.TextCode) ([]*response.BomHeaderResponse, error) {
	item, err := uc.resolveItem(ctx, code)
	if err != nil {
		return nil, err
	}
	hs, err := uc.repo.ListByItem(ctx, int64(item.Code))
	if err != nil {
		return nil, err
	}
	out := make([]*response.BomHeaderResponse, 0, len(hs))
	for _, h := range hs {
		out = append(out, toResponse(h, item))
	}
	return out, nil
}

// UpdateStatus movimenta o cabeçalho no ciclo de aprovação
// (RASCUNHO → APROVADO → OBSOLETO).
func (uc *BomHeaderUseCase) UpdateStatus(ctx context.Context, dto request.UpdateBomHeaderStatusDTO) (*response.BomHeaderResponse, error) {
	status := strings.ToUpper(strings.TrimSpace(dto.Status))
	if !entity.ValidStatus(status) {
		return nil, errorsuc.NewValidationError("situação inválida: use DRAFT (rascunho), APPROVED (aprovado) ou OBSOLETE (obsoleto)")
	}
	if dto.ID <= 0 {
		return nil, errorsuc.NewValidationError("informe o cabeçalho de estrutura a alterar")
	}
	updated, err := uc.repo.UpdateStatus(ctx, dto.ID, status)
	if err != nil {
		return nil, err
	}
	return toResponse(updated, uc.itemOf(ctx, updated.ItemCode)), nil
}

// legacyItemFinder é a fatia do repositório de itens usada para devolver o
// código de negócio na resposta a partir da chave legada gravada no cabeçalho.
type legacyItemFinder interface {
	FindItemByCode(context.Context, valueobject.ItemCode) (*itementity.Item, error)
}

// itemOf resolve o item pela chave legada só para enriquecer a resposta; a
// ausência não impede a operação.
func (uc *BomHeaderUseCase) itemOf(ctx context.Context, legacy int64) *itementity.Item {
	finder, ok := uc.items.(legacyItemFinder)
	if !ok || legacy <= 0 {
		return nil
	}
	item, err := finder.FindItemByCode(ctx, valueobject.ItemCode(legacy))
	if err != nil {
		return nil
	}
	return item
}
