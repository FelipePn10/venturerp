package ports

import "context"

// PreferredSupplierProvider resolves the preferred supplier for an item, used by
// the "Geração de Pedidos a partir de Solicitações" to default the supplier per
// item. Implemented by item_supplier_uc.
type PreferredSupplierProvider interface {
	// GetPreferredSupplier returns the lowest-ranking active supplier code for the
	// item, and whether one is registered.
	GetPreferredSupplier(ctx context.Context, itemCode int64) (supplierCode int64, found bool, err error)
}
