package configurator_uc

import (
	"context"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
)

// O item tem duas chaves: o código de negócio, que o usuário digita e que a API
// devolve como `code`, e a chave interna de `items.code`, usada pelas tabelas
// cfg_*. Sem tradução entre as duas, as características gravadas pela tela do
// configurador ficavam órfãs: o painel da estrutura procurava pela chave interna
// e nunca encontrava nada. Toda entrada de código de item passa por aqui.
func (uc *ConfiguratorUseCase) ResolveItemCode(ctx context.Context, code request.TextCode) (int64, error) {
	if uc.Items == nil {
		// Sem repositório de itens, cai no comportamento antigo (chave direta).
		parsed, err := strconv.ParseInt(code.String(), 10, 64)
		if err != nil || parsed <= 0 {
			return 0, errorsuc.NewValidationError("o código do item informado é inválido")
		}
		return parsed, nil
	}
	item, err := itemresolution.Resolve(ctx, uc.Items, code)
	if err != nil {
		return 0, err
	}
	return int64(item.Code), nil
}

// ResolveNumericItemCode traduz um código que chegou como número no corpo da
// requisição — os DTOs antigos ainda usam int64.
func (uc *ConfiguratorUseCase) ResolveNumericItemCode(ctx context.Context, code int64) (int64, error) {
	if code <= 0 {
		return 0, errorsuc.NewValidationError("o código do item é obrigatório")
	}
	return uc.ResolveItemCode(ctx, request.TextCode(strconv.FormatInt(code, 10)))
}
