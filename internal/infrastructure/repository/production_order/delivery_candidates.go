package production_order

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *ProductionOrderRepositoryPGX) ListDeliveryCandidates(ctx context.Context, filter production_order_uc.DeliveryCandidateFilter) ([]production_order_uc.DeliveryCandidate, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT po.id,po.order_number,po.employee_id,po.origin_type,po.item_code,
		COALESCE(item.pdm_description_technique,''),po.mask,po.planned_qty,COALESCE(delivery.delivered,0),
		GREATEST(po.planned_qty-COALESCE(delivery.delivered,0),0),po.warehouse_id,po.start_date,po.end_date
		FROM production_orders po LEFT JOIN items item ON item.code=po.item_code
		LEFT JOIN LATERAL(SELECT SUM(quantity) delivered FROM production_deliveries d WHERE d.enterprise_id=$1 AND d.production_order_id=po.id)delivery ON TRUE
		WHERE po.enterprise_id=$1 AND po.is_active AND po.status IN ('OPEN','RELEASED','IN_PROGRESS')
		AND ($2::bigint IS NULL OR po.order_number >= $2) AND ($3::bigint IS NULL OR po.order_number <= $3)
		AND ($4::bigint IS NULL OR po.item_code >= $4) AND ($5::bigint IS NULL OR po.item_code <= $5)
		AND ($6::bigint IS NULL OR po.employee_id >= $6) AND ($7::bigint IS NULL OR po.employee_id <= $7)
		AND ($8::date IS NULL OR po.start_date >= $8) AND ($9::date IS NULL OR po.end_date <= $9)
		AND ($10='' OR po.origin_type=$10) ORDER BY po.order_number`, enterpriseID, filter.OrderFrom, filter.OrderTo,
		filter.ItemFrom, filter.ItemTo, filter.PlannerFrom, filter.PlannerTo, filter.From, filter.To, filter.OrderType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []production_order_uc.DeliveryCandidate{}
	for rows.Next() {
		var row production_order_uc.DeliveryCandidate
		if err := rows.Scan(&row.ID, &row.OrderNumber, &row.Planner, &row.OrderType, &row.ItemCode, &row.Description, &row.Mask, &row.Planned, &row.Delivered, &row.Pending, &row.WarehouseID, &row.StartDate, &row.EndDate); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
