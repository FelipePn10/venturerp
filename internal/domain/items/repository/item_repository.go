package repository

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

var ErrNotFound = domainerr.NewNotFound("item não encontrado")
var ErrInvalidReference = domainerr.NewValidation("a pasta do item contém uma referência desconhecida")
var ErrConflict = domainerr.NewConflict("já existe um item com este código na empresa")

type ItemRepository interface {
	Create(ctx context.Context, item *entity.Item) (*entity.Item, error)
	UpdateFolders(ctx context.Context, item *entity.Item) (*entity.Item, error)
	FindItemByCode(ctx context.Context, code valueobject.ItemCode) (*entity.Item, error)
	ListAll(ctx context.Context) ([]*entity.Item, error)
	ListAllWithMasks(ctx context.Context) ([]entity.ItemWithMasks, error)
}
