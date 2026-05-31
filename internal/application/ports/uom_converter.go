package ports

import "context"

// UOMConverter converts quantities/prices between units of measure for an item,
// using the "Cadastro de Conversões por Item". Implemented by item_conversion_uc
// and consumed by the Purchase Order flow (UM compra ↔ estoque).
type UOMConverter interface {
	// Factor returns f where 1 fromUOM = f × toUOM for the item. Returns
	// found=false when no direct or inverse conversion is registered.
	Factor(ctx context.Context, itemCode int64, fromUOM, toUOM string) (factor float64, found bool, err error)
	// ConvertQuantity converts a quantity expressed in fromUOM to toUOM.
	ConvertQuantity(ctx context.Context, itemCode int64, qty float64, fromUOM, toUOM string) (float64, bool, error)
	// ConvertUnitPrice converts a unit price expressed per fromUOM to per toUOM.
	ConvertUnitPrice(ctx context.Context, itemCode int64, price float64, fromUOM, toUOM string) (float64, bool, error)
}
