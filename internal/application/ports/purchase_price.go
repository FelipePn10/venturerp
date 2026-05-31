package ports

import "context"

// PurchasePriceProvider resolves the unit price of an item from a purchase price
// table. Implemented by purchase_price_uc and consumed by the Purchase Order flow
// (the item price defaults from the table when one is set).
type PurchasePriceProvider interface {
	// GetItemPrice returns the price and UOM for an item in the table identified
	// by tableCode. Prefers a supplier-specific row when supplierCode is set.
	GetItemPrice(ctx context.Context, tableCode, itemCode int64, supplierCode *int64) (price float64, uom string, found bool, err error)
}
