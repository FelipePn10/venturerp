package production_order

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/jackc/pgx/v5"
)

// createThirdPartyOrdersTx keeps manual/MRP production-order creation and its
// external service orders in the same commit. The unique OF+route-operation key
// makes retries safe; the tenant advisory lock serializes the human-readable code.
func createThirdPartyOrdersTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, order *entity.ProductionOrder) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('third_party_service_orders:' || $1::bigint::text,0))`, enterpriseID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `WITH external_operations AS (
		SELECT ro.id route_operation_id,ro.operation_id,
			COALESCE((SELECT supplier.code FROM suppliers supplier
				WHERE supplier.id=COALESCE(ro.supplier_id,op.supplier_id)
				   OR supplier.code=COALESCE(ro.supplier_id,op.supplier_id)
				ORDER BY (supplier.id=COALESCE(ro.supplier_id,op.supplier_id)) DESC LIMIT 1),
				COALESCE(ro.supplier_id,op.supplier_id)) supplier_code,
			COALESCE(ro.service_item_code,op.service_item_code) service_item_code,
			COALESCE(ro.lead_time_days,op.lead_time_days,0) lead_days,
			COALESCE(ro.third_party_remittance,op.third_party_remittance,'DEMAND_ITEMS') remittance_type,
			ROW_NUMBER() OVER(ORDER BY ro.sequence,ro.id) sequence_number
		FROM manufacturing_routes route
		JOIN route_operations ro ON ro.route_id=route.id AND ro.is_active
		JOIN operations op ON op.id=ro.operation_id AND op.is_active AND op.origin IN ('EXTERNA','TERCEIROS')
		WHERE route.item_code=$3 AND route.is_active AND route.is_standard
	), code_base AS (
		SELECT COALESCE(MAX(code),0) value FROM third_party_service_orders WHERE enterprise_id=$1
	)
	INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,mask,
		supplier_code,service_item_code,uom,quantity,start_date,due_date,status,remittance_type,kanban,created_by)
	SELECT code_base.value+external_operations.sequence_number,$1,$2,route_operation_id,operation_id,$3,$4,supplier_code,
		service_item_code,COALESCE((SELECT warehouse_unit_of_measurement::text FROM items WHERE code=$3),'UN'),$5,
		COALESCE($6::date,CURRENT_DATE),COALESCE($7::date,COALESCE($6::date,CURRENT_DATE)+lead_days),'FIRM',
		CASE WHEN remittance_type IN ('DEMAND_ITEMS','ORDER_ITEM','GENERIC','NONE') THEN remittance_type ELSE 'DEMAND_ITEMS' END,
		EXISTS(SELECT 1 FROM kanban_cards k WHERE k.enterprise_id=$1 AND k.item_code=$3),$8
	FROM external_operations CROSS JOIN code_base
	ON CONFLICT(enterprise_id,production_order_id,route_operation_id) DO NOTHING`, enterpriseID, order.ID, order.ItemCode, order.Mask, order.PlannedQty, order.StartDate, order.EndDate, order.CreatedBy)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_service_order_history(enterprise_id,service_order_id,event_type,new_status,actor_id)
		SELECT $1,service_order.id,'CREATE',service_order.status,$3 FROM third_party_service_orders service_order
		WHERE service_order.enterprise_id=$1 AND service_order.production_order_id=$2
		AND NOT EXISTS(SELECT 1 FROM third_party_service_order_history history WHERE history.enterprise_id=$1 AND history.service_order_id=service_order.id AND history.event_type='CREATE')`, enterpriseID, order.ID, order.CreatedBy)
	return err
}
