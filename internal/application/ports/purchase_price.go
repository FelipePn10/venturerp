package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

// PurchasePriceProvider resolves the unit price of an item from a purchase price
// table. Implemented by purchase_price_uc and consumed by the Purchase Order flow
// (the item price defaults from the table when one is set).
type PurchasePriceProvider interface {
	// GetItemPrice devolve o preço LÍQUIDO (descontos e acréscimos da VSUP0120
	// aplicados em cadeia) e a unidade da linha da tabela. Prefere a linha do
	// fornecedor. found=false só quando não há linha vigente; falha de leitura
	// volta como erro (antes virava "sem preço" e o pedido saía a zero).
	GetItemPrice(ctx context.Context, tableCode, itemCode int64, supplierCode *int64) (price decimal.Decimal, uom string, found bool, err error)
}
