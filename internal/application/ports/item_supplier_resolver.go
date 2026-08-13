package ports

import "context"

type SupplierItemResolution struct {
	LinkID, ItemCode int64
	Strategy         string
}
type ItemSupplierResolver interface {
	ResolveExternal(context.Context, int64, string, string) (*SupplierItemResolution, error)
}
