package delivery_reschedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	reschedulerepo "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

func (r *DeliveryRescheduleRepositorySQLC) Preview(ctx context.Context, orderCode int64) ([]reschedulerepo.PlanningItem, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if r.pool == nil {
		return nil, errors.New("repositório de planejamento não configurado")
	}
	rows, err := r.pool.Query(ctx, `SELECT soi.code,soi.item_code,soi.sequence,soi.requested_qty::text,soi.attended_qty::text,soi.cancelled_qty::text,GREATEST(soi.requested_qty-soi.attended_qty-soi.cancelled_qty,0)::text,soi.delivery_date,soi.delivery_date_firm FROM sales_order_items soi JOIN sales_orders so ON so.code=soi.sales_order_code WHERE so.code=$1 AND so.enterprise_code=$2 AND soi.is_active ORDER BY soi.sequence`, orderCode, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []reschedulerepo.PlanningItem
	for rows.Next() {
		var v reschedulerepo.PlanningItem
		if err := rows.Scan(&v.SalesOrderItemCode, &v.ItemCode, &v.Sequence, &v.RequestedQty, &v.AttendedQty, &v.CancelledQty, &v.OpenQty, &v.CurrentDate, &v.FirmDate); err != nil {
			return nil, err
		}
		v.ReservedQty, v.IndependentDemandQty, v.InvoicedQty = "0", "0", "0"
		_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantity),0)::text FROM stock_reservations WHERE reference_type='SALES_ORDER' AND reference_code=$1 AND item_code=$2 AND status='ACTIVE'`, orderCode, int64(v.ItemCode)).Scan(&v.ReservedQty)
		_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantity-delivered_qty),0)::text FROM sales_order_demands WHERE sales_order_code=$1 AND item_code=$2 AND enterprise_id=$3 AND is_active`, orderCode, int64(v.ItemCode), enterpriseID).Scan(&v.IndependentDemandQty)
		_ = r.pool.QueryRow(ctx, `SELECT COUNT(*),COUNT(*) FILTER(WHERE po.is_firm),COALESCE(MAX(po.end_date),MAX(po.need_date)) FROM planned_orders po WHERE po.enterprise_id=$1 AND po.item_code=$2 AND po.is_active`, enterpriseID, int64(v.ItemCode)).Scan(&v.PlannedOrderCount, &v.FirmOrderCount, &v.SuggestedDate)
		_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM purchase_order_items poi JOIN purchase_orders po ON po.code=poi.purchase_order_code WHERE po.enterprise_code=$1 AND poi.item_code=$2 AND poi.is_active AND po.is_active AND po.status NOT IN ('CANCELLED','CLOSED')`, enterpriseID, int64(v.ItemCode)).Scan(&v.PurchaseOrderCount)
		_ = r.pool.QueryRow(ctx, `SELECT COUNT(*),MAX(s.status) FROM shipment_items si JOIN shipments s ON s.id=si.shipment_id WHERE s.sales_order_code=$1 AND si.sales_order_item_code=$2 AND s.status<>'CANCELLED'`, orderCode, v.SalesOrderItemCode).Scan(&v.ShipmentCount, &v.ShipmentStatus)
		_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(fei.quantity),0)::text FROM fiscal_exit_items fei JOIN fiscal_exits fe ON fe.id=fei.fiscal_exit_id WHERE fe.sales_order_code=$1 AND fei.item_code=$2 AND fe.is_active AND fe.status NOT IN ('DRAFT','CANCELLED','REJECTED')`, orderCode, int64(v.ItemCode)).Scan(&v.InvoicedQty)
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM capacity_requirements cr JOIN planned_orders po ON po.plan_code=cr.plan_code WHERE po.enterprise_id=$1 AND po.item_code=$2 AND cr.load_pct>100)`, enterpriseID, int64(v.ItemCode)).Scan(&v.CRPOverloaded)
		_ = r.pool.QueryRow(ctx, `SELECT MIN(ps.scheduled_end)::date FROM production_sequences ps JOIN production_orders prod ON prod.id=ps.production_order_id JOIN planned_orders po ON po.id=prod.planned_order_id WHERE po.enterprise_id=$1 AND po.item_code=$2`, enterpriseID, int64(v.ItemCode)).Scan(&v.APSDate)
		v.CanReschedule = v.OpenQty != "0" && v.InvoicedQty == "0" && (v.ShipmentStatus == nil || *v.ShipmentStatus != "SHIPPED")
		switch {
		case !v.CanReschedule:
			v.Severity = "BLOCK"
			v.SuggestionSource = "FISCAL"
			v.Justification = "item atendido, cancelado, faturado ou despachado não pode ser reprogramado"
		case v.CRPOverloaded:
			v.Severity = "WARNING"
			v.SuggestionSource = "CRP"
			v.Justification = "capacidade sobrecarregada; confirme a nova data com o planejamento"
		case v.APSDate != nil:
			v.Severity = "INFO"
			v.SuggestionSource = "APS"
			v.SuggestedDate = v.APSDate
			v.Justification = "data sugerida pelo sequenciamento APS"
		case v.SuggestedDate != nil:
			v.Severity = "INFO"
			v.SuggestionSource = "MRP"
			v.Justification = "data sugerida pela necessidade planejada"
		default:
			v.Severity = "INFO"
			v.SuggestionSource = "ATP"
			v.SuggestedDate = v.CurrentDate
			v.Justification = "sem sinal de planejamento posterior; mantida a data atual"
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, reschedulerepo.ErrOrderNotFound
	}
	return out, nil
}

func (r *DeliveryRescheduleRepositorySQLC) CreateBatch(ctx context.Context, c reschedulerepo.BatchCommand) (*reschedulerepo.BatchResult, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if r.pool == nil {
		return nil, errors.New("repositório de planejamento não configurado")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var existingHash string
	err = tx.QueryRow(ctx, `SELECT payload_hash FROM delivery_reschedule_batches WHERE enterprise_code=$1 AND idempotency_key=$2 FOR UPDATE`, enterpriseID, c.IdempotencyKey).Scan(&existingHash)
	if err == nil {
		if existingHash != c.PayloadHash {
			return nil, reschedulerepo.ErrBatchConflict
		}
		rows, e := tx.Query(ctx, `SELECT code FROM delivery_reschedules WHERE enterprise_code=$1 AND batch_id=(SELECT id FROM delivery_reschedule_batches WHERE enterprise_code=$1 AND idempotency_key=$2) ORDER BY code`, enterpriseID, c.IdempotencyKey)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		result := &reschedulerepo.BatchResult{Replayed: true}
		for rows.Next() {
			var code int64
			if e = rows.Scan(&code); e != nil {
				return nil, e
			}
			result.Codes = append(result.Codes, code)
		}
		return result, rows.Err()
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sales_orders WHERE code=$1 AND enterprise_code=$2 AND is_active)`, c.SalesOrderCode, enterpriseID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, reschedulerepo.ErrOrderNotFound
	}
	if _, err = tx.Exec(ctx, `INSERT INTO delivery_reschedule_batches(id,enterprise_code,sales_order_code,idempotency_key,payload_hash,created_by) VALUES($1,$2,$3,$4,$5,$6)`, c.ID, enterpriseID, c.SalesOrderCode, c.IdempotencyKey, c.PayloadHash, c.CreatedBy); err != nil {
		return nil, err
	}
	result := &reschedulerepo.BatchResult{}
	for _, line := range c.Lines {
		var current time.Time
		var open string
		err = tx.QueryRow(ctx, `SELECT COALESCE(soi.delivery_date,so.delivery_date,CURRENT_DATE),GREATEST(soi.requested_qty-soi.attended_qty-soi.cancelled_qty,0)::text FROM sales_order_items soi JOIN sales_orders so ON so.code=soi.sales_order_code WHERE soi.sales_order_code=$1 AND soi.code=$2 AND soi.item_code=$3 AND so.enterprise_code=$4 AND soi.is_active FOR UPDATE`, c.SalesOrderCode, line.SalesOrderItemCode, int64(line.ItemCode), enterpriseID).Scan(&current, &open)
		if err != nil || open == "0" || !sameDate(current, line.OldDate) {
			return nil, fmt.Errorf("%w: item %d não possui saldo ou a data atual foi alterada", reschedulerepo.ErrLinePrecondition, line.ItemCode)
		}
		var invoiced bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fiscal_exit_items fei JOIN fiscal_exits fe ON fe.id=fei.fiscal_exit_id WHERE fe.sales_order_code=$1 AND fei.item_code=$2 AND fe.is_active AND fe.status NOT IN ('DRAFT','CANCELLED','REJECTED'))`, c.SalesOrderCode, int64(line.ItemCode)).Scan(&invoiced); err != nil {
			return nil, err
		}
		if invoiced {
			return nil, fmt.Errorf("%w: item %d possui nota fiscal autorizada e exige tratamento fiscal", reschedulerepo.ErrLinePrecondition, line.ItemCode)
		}
		var shipped bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shipment_items si JOIN shipments s ON s.id=si.shipment_id WHERE s.sales_order_code=$1 AND si.sales_order_item_code=$2 AND s.status='SHIPPED')`, c.SalesOrderCode, line.SalesOrderItemCode).Scan(&shipped); err != nil {
			return nil, err
		}
		if shipped {
			return nil, fmt.Errorf("%w: item %d já foi despachado e não pode ser reprogramado", reschedulerepo.ErrLinePrecondition, line.ItemCode)
		}
		var code int64
		err = tx.QueryRow(ctx, `INSERT INTO delivery_reschedules(sales_order_code,item_code,old_date,new_date,reason,created_by,enterprise_code,batch_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING code`, c.SalesOrderCode, int64(line.ItemCode), line.OldDate, line.NewDate, line.Reason, c.CreatedBy, enterpriseID, c.ID).Scan(&code)
		if err != nil {
			return nil, err
		}
		result.Codes = append(result.Codes, code)
		if _, err = tx.Exec(ctx, `UPDATE sales_order_items SET delivery_date=$1,updated_at=NOW() WHERE sales_order_code=$2 AND code=$3`, line.NewDate, c.SalesOrderCode, line.SalesOrderItemCode); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE sales_order_demands SET delivery_date=$1,updated_at=NOW() WHERE enterprise_id=$2 AND sales_order_code=$3 AND item_code=$4 AND is_active AND NOT EXISTS(SELECT 1 FROM sales_order_items other WHERE other.sales_order_code=$3 AND other.item_code=$4 AND other.is_active AND other.code<>$5)`, line.NewDate, enterpriseID, c.SalesOrderCode, int64(line.ItemCode), line.SalesOrderItemCode); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE stock_reservations SET expiration_date=$1,updated_at=NOW() WHERE reference_type='SALES_ORDER' AND reference_code=$2 AND item_code=$3 AND status='ACTIVE' AND (reference_item_code=$4 OR (reference_item_code IS NULL AND NOT EXISTS(SELECT 1 FROM sales_order_items other WHERE other.sales_order_code=$2 AND other.item_code=$3 AND other.is_active AND other.code<>$4)))`, line.NewDate, c.SalesOrderCode, int64(line.ItemCode), line.SalesOrderItemCode); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

var _ reschedulerepo.PlanningRepository = (*DeliveryRescheduleRepositorySQLC)(nil)
