package aps

// Selection queries are intentionally dynamic SQL: optional array/range filters
// produce combinations that the legacy generated APS query set cannot express.
// Every statement anchors enterprise_id before applying user-selected filters.

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *APSRepositorySQLC) GetSelectedProductionOrders(ctx context.Context, f domainrepo.SequenceFilter) ([]domainrepo.OrderRow, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("APS selection repository requires database pool")
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT po.id,
		CASE WHEN po.priority ~ '^[0-9]+$' THEN po.priority::int WHEN UPPER(po.priority) IN ('ALTA','HIGH','URGENTE') THEN 1 WHEN UPPER(po.priority) IN ('BAIXA','LOW') THEN 9 ELSE 5 END,
		COALESCE(po.start_date::timestamptz,po.created_at)
		FROM production_orders po JOIN production_order_operations op ON op.production_order_id=po.id AND op.enterprise_id=po.enterprise_id
		WHERE po.enterprise_id=$1 AND po.status IN ('OPEN','RELEASED','IN_PROGRESS')
		AND ($2::bigint[] IS NULL OR po.id=ANY($2)) AND ($3::bigint[] IS NULL OR op.work_center_id=ANY($3))
		AND ($4::bigint[] IS NULL OR op.id=ANY($4))
		AND ($5::bigint[] IS NULL OR EXISTS(SELECT 1 FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code WHERE m.id=ANY($5) AND mt.id=op.work_center_id AND m.enterprise_id=$1))`,
		enterpriseID, nullIDs(f.OrderIDs), nullIDs(f.WorkCenterIDs), nullIDs(f.OperationIDs), nullIDs(f.MachineIDs))
	if err != nil {
		return nil, fmt.Errorf("listing selected production orders: %w", err)
	}
	defer rows.Close()
	out := []domainrepo.OrderRow{}
	for rows.Next() {
		var v domainrepo.OrderRow
		if err := rows.Scan(&v.ID, &v.Priority, &v.PlannedDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) GetSelectedOrderOperations(ctx context.Context, orderID int64, f domainrepo.SequenceFilter) ([]domainrepo.OpRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT op.id,op.sequence,op.work_center_id,op.planned_hours,op.setup_hours
		FROM production_order_operations op WHERE op.enterprise_id=$1 AND op.production_order_id=$2 AND op.status NOT IN ('DONE','SKIPPED')
		AND ($3::bigint[] IS NULL OR op.work_center_id=ANY($3)) AND ($4::bigint[] IS NULL OR op.id=ANY($4))
		AND ($5::bigint[] IS NULL OR EXISTS(SELECT 1 FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code WHERE m.id=ANY($5) AND mt.id=op.work_center_id AND m.enterprise_id=$1)) ORDER BY op.sequence`,
		enterpriseID, orderID, nullIDs(f.WorkCenterIDs), nullIDs(f.OperationIDs), nullIDs(f.MachineIDs))
	if err != nil {
		return nil, fmt.Errorf("listing selected operations: %w", err)
	}
	defer rows.Close()
	out := []domainrepo.OpRow{}
	for rows.Next() {
		var v domainrepo.OpRow
		if err := rows.Scan(&v.ID, &v.Sequence, &v.WorkCenterID, &v.PlannedHours, &v.SetupHours); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) ListSequencingEvents(ctx context.Context, f domainrepo.SequenceFilter) ([]domainrepo.SequencingEventRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT 'SCRAP',po.id,po.order_number,NULL::bigint,op.work_center_id,op.id,d.destination_date::timestamptz,d.scrap_quantity::text,COALESCE(d.destination_kind,'')
		FROM production_order_scrap_destinations d JOIN production_orders po ON po.id=d.production_order_id AND po.enterprise_id=d.enterprise_id
		LEFT JOIN LATERAL (SELECT candidate.id,candidate.work_center_id FROM production_order_operations candidate
			WHERE candidate.production_order_id=po.id AND candidate.enterprise_id=po.enterprise_id
			AND ($3::bigint[] IS NULL OR candidate.work_center_id=ANY($3)) AND ($4::bigint[] IS NULL OR candidate.id=ANY($4))
			AND ($5::bigint[] IS NULL OR EXISTS(SELECT 1 FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code WHERE m.id=ANY($5) AND m.enterprise_id=$1 AND mt.id=candidate.work_center_id))
			ORDER BY candidate.sequence LIMIT 1) op ON TRUE
		WHERE d.enterprise_id=$1 AND ($2::bigint[] IS NULL OR po.id=ANY($2)) AND (($3::bigint[] IS NULL AND $4::bigint[] IS NULL AND $5::bigint[] IS NULL) OR op.id IS NOT NULL)
		UNION ALL SELECT 'DOWNTIME',0,0,mp.machine_id,mo.work_center_id,NULL,mo.scheduled_date::timestamptz,mo.estimated_hours::text,COALESCE(mo.notes,'')
		FROM maintenance_orders mo JOIN maintenance_plans mp ON mp.id=mo.plan_id JOIN machines m ON m.id=mp.machine_id
		WHERE m.enterprise_id=$1 AND ($3::bigint[] IS NULL OR mo.work_center_id=ANY($3)) AND ($5::bigint[] IS NULL OR mp.machine_id=ANY($5)) ORDER BY 7`,
		enterpriseID, nullIDs(f.OrderIDs), nullIDs(f.WorkCenterIDs), nullIDs(f.OperationIDs), nullIDs(f.MachineIDs))
	if err != nil {
		return nil, fmt.Errorf("listing sequencing events: %w", err)
	}
	defer rows.Close()
	out := []domainrepo.SequencingEventRow{}
	for rows.Next() {
		var v domainrepo.SequencingEventRow
		if err := rows.Scan(&v.EventType, &v.ProductionOrderID, &v.OrderNumber, &v.MachineID, &v.WorkCenterID, &v.OperationID, &v.EventAt, &v.Quantity, &v.Reason); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) ListSequencingResources(ctx context.Context) ([]domainrepo.SequencingResourceRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT m.id,m.code,m.name,mt.id,m.resource_group_id,m.is_active FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code
		LEFT JOIN manufacturing_sequencing_settings s ON s.enterprise_id=m.enterprise_id WHERE m.enterprise_id=$1 AND (NOT COALESCE(s.list_only_active_resources,true) OR m.is_active) ORDER BY m.code`, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.SequencingResourceRow{}
	for rows.Next() {
		var v domainrepo.SequencingResourceRow
		if err := rows.Scan(&v.ID, &v.Code, &v.Name, &v.WorkCenterID, &v.ResourceGroupID, &v.IsActive); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) ListSequencingView(ctx context.Context, f domainrepo.SequencingViewFilter) ([]*entity.ProductionSequence, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT ps.id,ps.production_order_id,ps.operation_id,ps.work_center_id,ps.sequence_position,ps.scheduled_start,ps.scheduled_end,ps.status,ps.created_at,ps.updated_at
		FROM production_sequences ps JOIN production_orders po ON po.id=ps.production_order_id AND po.enterprise_id=ps.enterprise_id
		LEFT JOIN items i ON i.code=po.item_code LEFT JOIN production_order_operations op ON op.id=ps.operation_id AND op.enterprise_id=ps.enterprise_id
		WHERE ps.enterprise_id=$1 AND ps.scheduled_start<$3 AND ps.scheduled_end>$2
		AND EXISTS(SELECT 1 FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code WHERE mt.id=ps.work_center_id AND m.enterprise_id=$1 AND m.resource_group_id=$4)
		AND ($5::bigint IS NULL OR po.order_number >= $5) AND ($6::bigint IS NULL OR po.order_number <= $6)
		AND ($7::bigint IS NULL OR EXISTS(SELECT 1 FROM machines m WHERE m.enterprise_id=$1 AND m.id>= $7 AND m.machine_type_code=(SELECT code FROM machine_types WHERE id=ps.work_center_id)))
		AND ($8::bigint IS NULL OR EXISTS(SELECT 1 FROM machines m WHERE m.enterprise_id=$1 AND m.id<= $8 AND m.machine_type_code=(SELECT code FROM machine_types WHERE id=ps.work_center_id)))
		AND ($9::bigint IS NULL OR ps.work_center_id >= $9) AND ($10::bigint IS NULL OR ps.work_center_id <= $10)
		AND ($11::bigint IS NULL OR i.planner_employee_code >= $11) AND ($12::bigint IS NULL OR i.planner_employee_code <= $12)
		ORDER BY ps.scheduled_start,po.order_number,ps.sequence_position`, enterpriseID, f.From, f.To, f.ResourceGroupID, f.FromOrder, f.ToOrder, f.FromMachine, f.ToMachine, f.FromWorkCenter, f.ToWorkCenter, f.FromPlanner, f.ToPlanner)
	if err != nil {
		return nil, fmt.Errorf("listing sequencing view: %w", err)
	}
	defer rows.Close()
	out := []*entity.ProductionSequence{}
	for rows.Next() {
		v := &entity.ProductionSequence{}
		var status string
		if err := rows.Scan(&v.ID, &v.ProductionOrderID, &v.OperationID, &v.WorkCenterID, &v.SequencePosition, &v.ScheduledStart, &v.ScheduledEnd, &status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Status = entity.SequenceStatus(status)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) ListAvailabilityWindows(ctx context.Context, workCenterID int64, machineIDs []int64, from, to time.Time) ([]domainrepo.AvailabilityWindow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT day::date+COALESCE(i.start_time,TIME '00:00'),day::date+COALESCE(i.end_time,TIME '00:00')+CASE WHEN i.id IS NULL THEN make_interval(secs => ROUND(mt.capacity_hours*3600)::int) ELSE INTERVAL '0' END FROM generate_series($3::date,$4::date,'1 day') day
		JOIN machines m ON m.enterprise_id=$1 AND m.is_active JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.id=$2
		LEFT JOIN machine_calendar_intervals i ON i.calendar_id=m.calendar_id AND i.weekday=EXTRACT(DOW FROM day)::int
		WHERE ($5::bigint[] IS NULL OR m.id=ANY($5)) AND (i.id IS NOT NULL OR (m.calendar_id IS NULL AND EXTRACT(ISODOW FROM day)<6)) AND NOT EXISTS(SELECT 1 FROM maintenance_plans p JOIN maintenance_orders o ON o.plan_id=p.id WHERE p.machine_id=m.id AND o.is_active AND o.status IN ('PLANNED','IN_PROGRESS') AND o.scheduled_date=day::date)
		ORDER BY 1,2`, enterpriseID, workCenterID, from, to, nullIDs(machineIDs))
	if err != nil {
		return nil, fmt.Errorf("listing machine availability: %w", err)
	}
	defer rows.Close()
	out := []domainrepo.AvailabilityWindow{}
	for rows.Next() {
		var v domainrepo.AvailabilityWindow
		if err := rows.Scan(&v.Start, &v.End); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) ListCandidateMachines(ctx context.Context, workCenterID int64, machineIDs []int64) ([]domainrepo.MachineCandidate, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT m.id,mt.capacity_hours FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code WHERE m.enterprise_id=$1 AND mt.id=$2 AND m.is_active AND ($3::bigint[] IS NULL OR m.id=ANY($3)) ORDER BY m.is_critical DESC,m.code`, enterpriseID, workCenterID, nullIDs(machineIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.MachineCandidate{}
	for rows.Next() {
		var v domainrepo.MachineCandidate
		if err := rows.Scan(&v.ID, &v.CapacityHours); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *APSRepositorySQLC) ListMachineDowntimeWindows(ctx context.Context, machineID int64, from, to time.Time) ([]domainrepo.AvailabilityWindow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT starts_at,ends_at FROM machine_downtimes WHERE enterprise_id=$1 AND machine_id=$2 AND starts_at<$4 AND ends_at>$3 ORDER BY starts_at`, enterpriseID, machineID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.AvailabilityWindow{}
	for rows.Next() {
		var v domainrepo.AvailabilityWindow
		if err := rows.Scan(&v.Start, &v.End); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func nullIDs(v []int64) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

var _ = time.Time{}
