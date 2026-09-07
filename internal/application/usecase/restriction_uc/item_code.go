package restriction_uc

import (
	"context"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
)

// O item tem duas chaves: o código de negócio, que o usuário digita, e a chave
// interna de `items.code`, que é a que o configurador usa ao consultar as
// restrições. Gravar a restrição com o código de negócio a tornava invisível
// para o configurador — a regra existia e não bloqueava nada.
func resolverCodigoDoItem(ctx context.Context, items any, code *int64) (*int64, error) {
	if items == nil || code == nil || *code <= 0 {
		return code, nil
	}
	item, err := itemresolution.Resolve(ctx, items, request.TextCode(strconv.FormatInt(*code, 10)))
	if err != nil {
		return nil, err
	}
	interno := int64(item.Code)
	return &interno, nil
}
