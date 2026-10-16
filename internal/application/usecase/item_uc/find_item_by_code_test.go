package item_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type findItemAuth struct{ ports.AuthService }

func (findItemAuth) FindItemByCode(context.Context) bool { return true }

type missingItemRepository struct{}

func (missingItemRepository) UpdateFolders(context.Context, *entity.Item) (*entity.Item, error) {
	return nil, repository.ErrNotFound
}

func (missingItemRepository) Create(context.Context, *entity.Item) (*entity.Item, error) {
	return nil, errors.New("unexpected Create call")
}
func (missingItemRepository) FindItemByCode(context.Context, valueobject.ItemCode) (*entity.Item, error) {
	return nil, repository.ErrNotFound
}

func (missingItemRepository) FindItemByBusinessCode(context.Context, valueobject.BusinessCode) (*entity.Item, error) {
	return nil, repository.ErrNotFound
}
func (missingItemRepository) ListAll(context.Context) ([]*entity.Item, error) {
	return nil, errors.New("unexpected ListAll call")
}
func (missingItemRepository) ListAllWithMasks(context.Context) ([]entity.ItemWithMasks, error) {
	return nil, errors.New("unexpected ListAllWithMasks call")
}

func TestFindItemByCodeTranslatesRepositoryNotFound(t *testing.T) {
	uc := NewFindItemByCode(missingItemRepository{}, findItemAuth{})
	_, err := uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: "TEA452-0"})
	if !errors.Is(err, errorsuc.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

type translatedLegacyItemRepository struct{ missingItemRepository }

func (translatedLegacyItemRepository) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*entity.Item, error) {
	if code == 202 {
		return &entity.Item{Code: code, BusinessCode: "TP-01001-A"}, nil
	}
	return nil, repository.ErrNotFound
}

func TestFindItemByCodeAcceptsLegacyKeyProducedByCompatibilityMiddleware(t *testing.T) {
	uc := NewFindItemByCode(translatedLegacyItemRepository{}, findItemAuth{})

	item, err := uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: "202"})
	if err != nil {
		t.Fatalf("buscar chave traduzida: %v", err)
	}
	if item.Code != "TP-01001-A" || item.LegacyCode != 202 {
		t.Fatalf("resposta inesperada: code=%q legacy=%d", item.Code, item.LegacyCode)
	}
}

type descricoesFalsas map[string][]ItemPorDescricao

func (d descricoesFalsas) ItensPorDescricao(_ context.Context, texto string, _ int) ([]ItemPorDescricao, error) {
	return d[texto], nil
}

// O mesmo campo aceita o código ou a descrição: um item só abre direto; vários
// pedem o código, listando as opções.
func TestFindItemByCodeAceitaDescricao(t *testing.T) {
	repo := newRefItems()
	uc := NewFindItemByCode(repo, findItemAuth{})
	uc.Descricao = descricoesFalsas{
		"Caixa de papelão":    {{Codigo: "CX-PAPELAO", Descricao: "Caixa de papelão"}},
		"Produto":             {{Codigo: "PROD-01", Descricao: "Produto 01"}, {Codigo: "MOD-BASE", Descricao: "Modelo base"}},
		"BASE 195 X 83 X 3/8": nil,
	}

	item, err := uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: "Caixa de papelão"})
	if err != nil || item.Code != "CX-PAPELAO" {
		t.Fatalf("descrição única: item=%+v err=%v", item, err)
	}
	_, err = uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: "Produto"})
	if v, ok := errorsuc.AsValidation(err); !ok || v == nil {
		t.Fatalf("várias descrições: erro=%v, quer validação listando os códigos", err)
	}
	if item, err := uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: "PROD-01"}); err != nil || item.Code != "PROD-01" {
		t.Fatalf("código continua valendo: item=%+v err=%v", item, err)
	}
	if _, err := uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: "BASE 195 X 83 X 3/8"}); err == nil {
		t.Fatal("texto sem item nenhum deveria dar erro")
	}
}
