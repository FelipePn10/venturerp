package itemresolution

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type businessCodeFinder interface {
	FindItemByBusinessCode(context.Context, valueobject.BusinessCode) (*itementity.Item, error)
}

type legacyCodeFinder interface {
	FindItemByCode(context.Context, valueobject.ItemCode) (*itementity.Item, error)
}

type maskLister interface {
	ListAllWithMasks(context.Context) ([]itementity.ItemWithMasks, error)
}

// Resolve converts the public commercial code into the immutable legacy key.
// Numeric JSON remains accepted during the compatibility window. A numeric
// string is first treated as a legitimate business code and only then as the
// legacy key, avoiding ambiguity for codes such as "0007".
func Resolve(ctx context.Context, repository any, raw request.TextCode) (*itementity.Item, error) {
	code := strings.TrimSpace(raw.String())
	if code == "" {
		return nil, errorsuc.NewValidationError("o código do item é obrigatório")
	}
	businessCode, err := valueobject.NewBusinessCode(code)
	if err != nil {
		return nil, errorsuc.NewValidationError("o código do item informado é inválido")
	}
	if finder, ok := repository.(businessCodeFinder); ok {
		item, findErr := finder.FindItemByBusinessCode(ctx, businessCode)
		if findErr == nil {
			return item, nil
		}
		if !errors.Is(findErr, itemrepo.ErrNotFound) {
			return nil, findErr
		}
	}
	legacy, parseErr := strconv.ParseInt(code, 10, 64)
	if parseErr == nil && legacy > 0 {
		if finder, ok := repository.(legacyCodeFinder); ok {
			item, findErr := finder.FindItemByCode(ctx, valueobject.ItemCode(legacy))
			if findErr == nil {
				return item, nil
			}
			if !errors.Is(findErr, itemrepo.ErrNotFound) {
				return nil, findErr
			}
		}
	}
	return nil, errorsuc.NewValidationError("item não encontrado na empresa autenticada")
}

func ValidateMask(ctx context.Context, repository any, itemCode int64, mask string) error {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return nil
	}
	lister, ok := repository.(maskLister)
	if !ok {
		return errorsuc.NewValidationError("não foi possível validar a máscara do item")
	}
	items, err := lister.ListAllWithMasks(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Item == nil || int64(item.Item.Code) != itemCode {
			continue
		}
		for _, registered := range item.Masks {
			if strings.EqualFold(strings.TrimSpace(registered.Mask), mask) {
				return nil
			}
		}
		break
	}
	return errorsuc.NewValidationError("a máscara informada não está cadastrada para o item na empresa autenticada")
}
