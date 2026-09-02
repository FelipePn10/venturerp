package routing_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
)

func resolveOptionalItemCode(ctx context.Context, items any, code *request.TextCode) (*int64, error) {
	if code == nil || code.String() == "" {
		return nil, nil
	}
	item, err := itemresolution.Resolve(ctx, items, *code)
	if err != nil {
		return nil, err
	}
	legacy := int64(item.Code)
	return &legacy, nil
}
