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
