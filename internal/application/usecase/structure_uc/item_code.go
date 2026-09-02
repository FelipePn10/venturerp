package structure_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func resolveItemCode(ctx context.Context, items any, code request.TextCode) (int64, error) {
	item, err := itemresolution.Resolve(ctx, items, code)
	if err != nil {
		return 0, err
	}
	return int64(item.Code), nil
}

// resolveItem devolve o item completo da empresa autenticada a partir do código
// de negócio — usado quando a resposta precisa do código e da descrição.
func resolveItem(ctx context.Context, items any, code request.TextCode) (*itementity.Item, error) {
	return itemresolution.Resolve(ctx, items, code)
}
