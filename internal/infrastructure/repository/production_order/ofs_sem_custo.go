package production_order

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

// OFsSemCusto lista, em TODAS as empresas, as OFs não canceladas que já se
// moveram e não têm custo apurado: a fila da rotina que apura o custo das OFs
// antigas. Cada OF é apurada depois na sessão da empresa dela.
func (r *ProductionOrderRepositoryPGX) OFsSemCusto(ctx context.Context, max int) ([]entity.OFSemCusto, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.enterprise_id, e.code, o.created_by
		FROM production_orders o
		JOIN enterprise e ON e.id = o.enterprise_id
		WHERE o.status <> 'CANCELLED' AND o.is_active IS NOT FALSE
		  AND NOT EXISTS (SELECT 1 FROM production_order_costs c WHERE c.production_order_id = o.id)
		  AND (EXISTS (SELECT 1 FROM production_appointments a WHERE a.production_order_id = o.id)
		    OR EXISTS (SELECT 1 FROM production_order_operations op WHERE op.production_order_id = o.id
		               AND op.enterprise_id = o.enterprise_id AND COALESCE(op.actual_hours, 0) > 0)
		    OR EXISTS (SELECT 1 FROM stock_movements m WHERE m.enterprise_id = o.enterprise_id
		               AND m.reference_type = 'PRODUCTION_ORDER' AND m.reference_code = o.id))
		ORDER BY o.id
		LIMIT $1`, max)
	if err != nil {
		return nil, fmt.Errorf("listando as OFs sem custo apurado: %w", err)
	}
	defer rows.Close()
	var out []entity.OFSemCusto
	for rows.Next() {
		var of entity.OFSemCusto
		if err := rows.Scan(&of.ID, &of.EnterpriseID, &of.EnterpriseCode, &of.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, of)
	}
	return out, rows.Err()
}
