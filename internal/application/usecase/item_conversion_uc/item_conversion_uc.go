package item_conversion_uc

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	c, err := entity.NewItemUnitConversion(dto.ItemCode, dto.Mask, dto.FromUOM, dto.ToUOM, dto.Factor, dto.RoundingPercent, dto.ToleranceValue, dto.ToleranceType, dto.CreatedBy)
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
	return uc.ConvertQuantityConfigured(ctx, itemCode, "", qty, fromUOM, toUOM)
}

type configuredConversionRepository interface {
	GetConfigured(context.Context, int64, string, string, string) (*entity.ItemUnitConversion, error)
	AcceptsFractional(context.Context, int64) (bool, error)
}

func (uc *ItemConversionUseCase) FactorConfigured(ctx context.Context, itemCode int64, mask, fromUOM, toUOM string) (float64, bool, error) {
	from, to := strings.ToUpper(strings.TrimSpace(fromUOM)), strings.ToUpper(strings.TrimSpace(toUOM))
	if from == to {
		return 1, true, nil
	}
	repo, supportsPolicy := uc.repo.(configuredConversionRepository)
	if !supportsPolicy {
		return uc.Factor(ctx, itemCode, from, to)
	}
	conversion, inverse := resolveConfiguredConversion(ctx, repo, itemCode, mask, from, to)
	if conversion == nil {
		return 0, false, nil
	}
	if inverse {
		return 1 / conversion.Factor, true, nil
	}
	return conversion.Factor, true, nil
}

func resolveConfiguredConversion(ctx context.Context, repo configuredConversionRepository, itemCode int64, mask, from, to string) (*entity.ItemUnitConversion, bool) {
	conversion, _ := repo.GetConfigured(ctx, itemCode, mask, from, to)
	inverse := false
	if conversion == nil {
		conversion, _ = repo.GetConfigured(ctx, itemCode, mask, to, from)
		inverse = conversion != nil
	}
	if conversion == nil && mask != "" {
		conversion, _ = repo.GetConfigured(ctx, itemCode, "", from, to)
		inverse = false
		if conversion == nil {
			conversion, _ = repo.GetConfigured(ctx, itemCode, "", to, from)
			inverse = conversion != nil
		}
	}
	return conversion, inverse
}

func (uc *ItemConversionUseCase) ConvertQuantityConfigured(ctx context.Context, itemCode int64, mask string, qty float64, fromUOM, toUOM string) (float64, bool, error) {
	from, to := strings.ToUpper(strings.TrimSpace(fromUOM)), strings.ToUpper(strings.TrimSpace(toUOM))
	if from == to {
		return qty, true, nil
	}
	repo, supportsPolicy := uc.repo.(configuredConversionRepository)
	if !supportsPolicy {
		f, found, err := uc.Factor(ctx, itemCode, from, to)
		if err != nil || !found {
			return 0, found, err
		}
		return qty * f, true, nil
	}
	conversion, inverse := resolveConfiguredConversion(ctx, repo, itemCode, mask, from, to)
	if conversion == nil {
		return 0, false, nil
	}
	factor := conversion.Factor
	if inverse {
		factor = 1 / factor
	}
	converted := qty * factor
	fractional, err := repo.AcceptsFractional(ctx, itemCode)
	if err != nil {
		return 0, false, err
	}
	if fractional {
		return converted, true, nil
	}
	rounded := math.Round(converted)
	allowed := math.Abs(converted) * conversion.RoundingPercent / 100
	if conversion.ToleranceType == "PERCENT" {
		allowed += math.Abs(converted) * conversion.ToleranceValue / 100
	} else {
		allowed += conversion.ToleranceValue
	}
	if math.Abs(converted-rounded) > allowed {
		return 0, false, fmt.Errorf("converted quantity %.8f is fractional and exceeds rounding/tolerance policy", converted)
	}
	return rounded, true, nil
}

// ConvertUnitPrice: if 1 fromUOM = f toUOM, then price per toUOM = price / f.
func (uc *ItemConversionUseCase) ConvertUnitPrice(ctx context.Context, itemCode int64, price float64, fromUOM, toUOM string) (float64, bool, error) {
	f, found, err := uc.Factor(ctx, itemCode, fromUOM, toUOM)
	if err != nil || !found || f == 0 {
		return 0, found, err
	}
	return price / f, true, nil
}
