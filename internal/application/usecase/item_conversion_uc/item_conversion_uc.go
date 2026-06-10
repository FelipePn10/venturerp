package item_conversion_uc

import (
	"context"
	"errors"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
)

// ErrNoConversion is returned when no direct/inverse factor is registered.
var ErrNoConversion = errors.New("não existe fator de conversão cadastrado para o item — cadastre em Conversões por Item")

type ItemConversionUseCase struct {
	repo repository.ItemConversionRepository
}

func NewItemConversionUseCase(repo repository.ItemConversionRepository) *ItemConversionUseCase {
	return &ItemConversionUseCase{repo: repo}
}

func (uc *ItemConversionUseCase) Create(ctx context.Context, dto request.CreateItemConversionDTO) (*response.ItemUnitConversionResponse, error) {
	c, err := entity.NewItemUnitConversion(dto.ItemCode, dto.FromUOM, dto.ToUOM, dto.Factor, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, c)
	if err != nil {
		return nil, err
	}
	return toItemConversionResponse(created), nil
}

func (uc *ItemConversionUseCase) ListByItem(ctx context.Context, itemCode int64) ([]*response.ItemUnitConversionResponse, error) {
	list, err := uc.repo.ListByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toItemConversionResponses(list), nil
}

func (uc *ItemConversionUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// ─── ports.UOMConverter implementation ──────────────────────────────────────

// Factor returns f where 1 fromUOM = f × toUOM. Tries the direct conversion,
// then the inverse (1/factor). Same unit returns 1.
func (uc *ItemConversionUseCase) Factor(ctx context.Context, itemCode int64, fromUOM, toUOM string) (float64, bool, error) {
	from := strings.ToUpper(strings.TrimSpace(fromUOM))
	to := strings.ToUpper(strings.TrimSpace(toUOM))
	if from == to {
		return 1, true, nil
	}
	if c, err := uc.repo.Get(ctx, itemCode, from, to); err == nil && c != nil {
		return c.Factor, true, nil
	}
	if c, err := uc.repo.Get(ctx, itemCode, to, from); err == nil && c != nil && c.Factor != 0 {
		return 1 / c.Factor, true, nil
	}
	return 0, false, nil
}

func (uc *ItemConversionUseCase) ConvertQuantity(ctx context.Context, itemCode int64, qty float64, fromUOM, toUOM string) (float64, bool, error) {
	f, found, err := uc.Factor(ctx, itemCode, fromUOM, toUOM)
	if err != nil || !found {
		return 0, found, err
	}
	return qty * f, true, nil
}

// ConvertUnitPrice: if 1 fromUOM = f toUOM, then price per toUOM = price / f.
func (uc *ItemConversionUseCase) ConvertUnitPrice(ctx context.Context, itemCode int64, price float64, fromUOM, toUOM string) (float64, bool, error) {
	f, found, err := uc.Factor(ctx, itemCode, fromUOM, toUOM)
	if err != nil || !found || f == 0 {
		return 0, found, err
	}
	return price / f, true, nil
}
