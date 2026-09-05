package repository

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

var ErrNotFound = errors.New("item não encontrado")
var ErrInvalidReference = errors.New("a pasta do item contém uma referência desconhecida")
var ErrConflict = errors.New("já existe um item com este código na empresa")

type ItemRepository interface {
	Create(ctx context.Context, item *entity.Item) (*entity.Item, error)
	UpdateCommercialAccounting(ctx context.Context, item *entity.Item) (*entity.Item, error)
	FindItemByCode(ctx context.Context, code valueobject.ItemCode) (*entity.Item, error)
	ListAll(ctx context.Context) ([]*entity.Item, error)
	ListAllWithMasks(ctx context.Context) ([]entity.ItemWithMasks, error)
}
