package production_order

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// HasProductionOperations prevents starting an empty OF and verifies ownership
// through the parent production order in the same query.
func (r *ProductionOrderRepositoryPGX) HasProductionOperations(ctx context.Context, orderID int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM production_order_operations operation
		JOIN production_orders production_order ON production_order.id = operation.production_order_id
		WHERE operation.production_order_id = $1 AND production_order.enterprise_id = $2
	)`, orderID, enterpriseID).Scan(&exists)
	return exists, err
}
