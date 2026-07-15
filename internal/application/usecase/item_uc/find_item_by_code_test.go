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

func (missingItemRepository) Create(context.Context, *entity.Item) (*entity.Item, error) {
	return nil, errors.New("unexpected Create call")
}
func (missingItemRepository) FindItemByCode(context.Context, valueobject.ItemCode) (*entity.Item, error) {
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
	_, err := uc.Execute(context.Background(), request.FindItemByCodeDTO{Code: valueobject.ItemCode(10001)})
	if !errors.Is(err, errorsuc.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}
