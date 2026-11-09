package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

// UOMConverter converts quantities/prices between units of measure for an item,
// using the "Cadastro de Conversões por Item". Implemented by item_conversion_uc
// and consumed by the Purchase Order flow (UM compra ↔ estoque).
type UOMConverter interface {
	// Factor returns f where 1 fromUOM = f × toUOM for the item. Returns
	// found=false when no direct or inverse conversion is registered.
	Factor(ctx context.Context, itemCode int64, fromUOM, toUOM string) (factor decimal.Decimal, found bool, err error)
	// ConvertQuantity converts a quantity expressed in fromUOM to toUOM.
	ConvertQuantity(ctx context.Context, itemCode int64, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error)
	// ConvertUnitPrice converts a unit price expressed per fromUOM to per toUOM.
	ConvertUnitPrice(ctx context.Context, itemCode int64, price decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error)
}

// MaskedUOMConverter é opcional: quem também conhece a máscara (configurador)
// usa a conversão dela antes da do item. Quem só implementa UOMConverter
// continua funcionando, sem máscara.
type MaskedUOMConverter interface {
	ConvertQuantityConfigured(ctx context.Context, itemCode int64, mask string, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error)
}

// ConvertQuantityWithMask usa a máscara quando o conversor sabe lidar com ela.
func ConvertQuantityWithMask(ctx context.Context, c UOMConverter, itemCode int64, mask string, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	if m, ok := c.(MaskedUOMConverter); ok {
		return m.ConvertQuantityConfigured(ctx, itemCode, mask, qty, fromUOM, toUOM)
	}
	return c.ConvertQuantity(ctx, itemCode, qty, fromUOM, toUOM)
}
