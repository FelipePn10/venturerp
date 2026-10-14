package production_order

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// MaterialAvailability lê, para as OFs informadas, cada material a baixar com o
// saldo livre do depósito e o que as OFs abertas anteriores já vão consumir do
// mesmo item. "Anterior" é quem começa antes (sem início = fim da fila) ou, no
// mesmo dia, foi criada antes — é a fila que o almoxarifado atende.
func (r *ProductionOrderRepositoryPGX) MaterialAvailability(ctx context.Context, orderIDs []int64) ([]entity.MaterialAvailabilityLine, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.production_order_id, m.id, m.item_code, COALESCE(i.business_code::text, ''), COALESCE(i.name, ''),
		       COALESCE(m.mask, ''), COALESCE(m.uom, ''), m.warehouse_id, m.quantity, m.attended_quantity,
		       COALESCE((SELECT SUM(b.quantity - b.reserved_qty) FROM stock_balances b
		                  WHERE b.enterprise_id = $1 AND b.item_code = m.item_code
		                    AND COALESCE(b.mask, '') = COALESCE(m.mask, '')
		                    AND (m.warehouse_id IS NULL OR b.warehouse_id = m.warehouse_id)), 0),
		       COALESCE((SELECT SUM(GREATEST(o.quantity - o.attended_quantity, 0))
		                   FROM production_order_materials o
		                   JOIN production_orders op ON op.id = o.production_order_id AND op.enterprise_id = $1
		                  WHERE o.enterprise_id = $1 AND o.material_kind = 'DEMAND'
		                    AND o.item_code = m.item_code AND COALESCE(o.mask, '') = COALESCE(m.mask, '')
		                    AND (m.warehouse_id IS NULL OR o.warehouse_id IS NULL OR o.warehouse_id = m.warehouse_id)
		                    AND op.is_active AND op.status IN ('OPEN', 'RELEASED', 'IN_PROGRESS')
		                    AND op.id <> po.id
		                    AND (COALESCE(op.start_date, 'infinity'::date) < COALESCE(po.start_date, 'infinity'::date)
		                         OR (COALESCE(op.start_date, 'infinity'::date) = COALESCE(po.start_date, 'infinity'::date) AND op.id < po.id))), 0)
		  FROM production_order_materials m
		  JOIN production_orders po ON po.id = m.production_order_id AND po.enterprise_id = $1
		  LEFT JOIN items i ON i.code = m.item_code AND i.enterprise_id = $1
		 WHERE m.enterprise_id = $1 AND m.material_kind = 'DEMAND' AND po.id = ANY($2::bigint[])
		 ORDER BY m.production_order_id, m.id`, enterpriseID, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.MaterialAvailabilityLine{}
	for rows.Next() {
		var l entity.MaterialAvailabilityLine
		if err := rows.Scan(&l.ProductionOrderID, &l.MaterialID, &l.ItemCode, &l.BusinessCode, &l.Description,
			&l.Mask, &l.UOM, &l.WarehouseID, &l.Required, &l.Issued, &l.FreeStock, &l.CommittedEarlier); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
