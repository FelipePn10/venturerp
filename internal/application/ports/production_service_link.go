package ports

import "context"

// ProductionServiceLinker preserves traceability from the manufacturing order
// through the service purchase requisition to every generated service PO.
type ProductionServiceLinker interface {
	CurrentEnterpriseCode(ctx context.Context) (int64, error)
	LinkServiceRequisition(ctx context.Context, productionOrderID, requisitionCode int64) error
	LinkServicePurchaseOrder(ctx context.Context, requisitionItemID, purchaseOrderCode int64) error
}
