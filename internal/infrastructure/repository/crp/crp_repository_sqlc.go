package crp

import (
	"context"
	"fmt"
	capacity "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine_capacity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/crp/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/crp/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type CRPRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pools ...*pgxpool.Pool) domainrepo.CRPRepository {
	r := &CRPRepositorySQLC{q: q}
	if len(pools) > 0 {
		r.pool = pools[0]
	}
	return r
}

func (r *CRPRepositorySQLC) UpsertRequirement(ctx context.Context, req *entity.CapacityRequirement) (*entity.CapacityRequirement, error) {
	row, err := r.q.UpsertCapacityRequirement(ctx, sqlc.UpsertCapacityRequirementParams{
		PlanCode:       req.PlanCode,
		WorkCenterID:   req.WorkCenterID,
		ReqDate:        sqlc.ToPgDate(req.ReqDate),
		RequiredHours:  pgutil.ToPgNumericFromFloat64(req.RequiredHours),
		AvailableHours: pgutil.ToPgNumericFromFloat64(req.AvailableHours),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting capacity requirement: %w", err)
	}
	return crpRowToEntity(row), nil
}

func (r *CRPRepositorySQLC) ListByPlan(ctx context.Context, planCode int64) ([]*entity.CapacityRequirement, error) {
	rows, err := r.q.ListCRPByPlan(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("listing CRP for plan %d: %w", planCode, err)
	}
	return crpSlice(rows), nil
}

func (r *CRPRepositorySQLC) ListOverloadedByPlan(ctx context.Context, planCode int64) ([]*entity.CapacityRequirement, error) {
	rows, err := r.q.ListOverloadedCRPByPlan(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("listing overloaded CRP for plan %d: %w", planCode, err)
	}
	return crpSlice(rows), nil
}

func (r *CRPRepositorySQLC) ListByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.CapacityRequirement, error) {
	rows, err := r.q.ListCRPByWorkCenter(ctx, workCenterID, sqlc.ToPgDate(from), sqlc.ToPgDate(to))
	if err != nil {
		return nil, fmt.Errorf("listing CRP for work center %d: %w", workCenterID, err)
	}
	return crpSlice(rows), nil
}

func (r *CRPRepositorySQLC) DeleteByPlan(ctx context.Context, planCode int64) error {
	return r.q.DeleteCRPByPlan(ctx, planCode)
}

func (r *CRPRepositorySQLC) GetPlannedOrdersByPlan(ctx context.Context, planCode int64) ([]domainrepo.PlannedOrderRow, error) {
	if r.pool != nil {
		return r.plannedLoads(ctx, planCode)
	}
	rows, err := r.q.GetPlannedOrdersForCRP(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("fetching planned orders for CRP: %w", err)
	}
	out := make([]domainrepo.PlannedOrderRow, 0, len(rows))
	for _, row := range rows {
		po := domainrepo.PlannedOrderRow{
			ID:          row.ID,
			ItemCode:    row.ItemCode,
			Quantity:    row.Quantity,
			PlannedDate: sqlc.FromPgDate(row.PlannedDate),
		}
		if row.RouteID.Valid {
			v := row.RouteID.Int64
			po.RouteID = &v
		}
		out = append(out, po)
	}
	return out, nil
}

func (r *CRPRepositorySQLC) GetRouteOperationsByRoute(ctx context.Context, routeID int64) ([]domainrepo.RouteOpRow, error) {
	rows, err := r.q.GetRouteOpHoursForCRP(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching route operations for CRP: %w", err)
	}
	out := make([]domainrepo.RouteOpRow, 0, len(rows))
	for _, row := range rows {
		ro := domainrepo.RouteOpRow{EffHours: row.EffHours}
		if row.WorkCenterID.Valid {
			v := row.WorkCenterID.Int64
			ro.WorkCenterID = &v
		}
		out = append(out, ro)
	}
	return out, nil
}

func (r *CRPRepositorySQLC) GetMachineAvailableHoursPerDay(ctx context.Context, workCenterID int64) (float64, error) {
	if r.pool != nil {
		return capacity.Hours(ctx, r.pool, workCenterID, nil)
	}
	return r.q.GetMachineAvailableHours(ctx, workCenterID)
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func crpRowToEntity(row sqlc.DBCapacityRequirement) *entity.CapacityRequirement {
	reqHours := pgutil.FromPgNumericToFloat64(row.RequiredHours)
	availHours := pgutil.FromPgNumericToFloat64(row.AvailableHours)
	loadPct := pgutil.FromPgNumericToFloat64(row.LoadPct)
	return &entity.CapacityRequirement{
		ID:             row.ID,
		PlanCode:       row.PlanCode,
		WorkCenterID:   row.WorkCenterID,
		ReqDate:        sqlc.FromPgDate(row.ReqDate),
		RequiredHours:  reqHours,
		AvailableHours: availHours,
		LoadPct:        loadPct,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func crpSlice(rows []sqlc.DBCapacityRequirement) []*entity.CapacityRequirement {
	out := make([]*entity.CapacityRequirement, 0, len(rows))
	for _, row := range rows {
		out = append(out, crpRowToEntity(row))
	}
	return out
}

func (r *CRPRepositorySQLC) GetAvailableHoursOnDate(ctx context.Context, wc int64, day time.Time) (float64, error) {
	if r.pool == nil {
		return r.GetMachineAvailableHoursPerDay(ctx, wc)
	}
	return capacity.Hours(ctx, r.pool, wc, &day)
}

func (r *CRPRepositorySQLC) plannedLoads(ctx context.Context, plan int64) ([]domainrepo.PlannedOrderRow, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `WITH orders AS (
 SELECT po.id,po.item_code,po.mask,COALESCE(NULLIF(po.quantity_corrected,0),po.quantity)::double precision AS quantity,COALESCE(po.start_date,po.need_date) AS day,(SELECT m.id FROM machines m WHERE m.code=po.machine_code AND m.enterprise_id=po.enterprise_id) AS machine_id,po.production_time AS minutes
 FROM planned_orders po WHERE po.plan_code=$1 AND po.enterprise_id=$2 AND po.is_active AND po.status<>'CANCELLED' AND po.order_type::text='PRODUCTION' AND NOT EXISTS(SELECT 1 FROM mrp_machine_allocation_slots sl WHERE sl.suggestion_code=po.mrp_suggestion_code)
 UNION ALL
 SELECT -s.code,s.item_code,s.mask,s.quantity::double precision,COALESCE(s.start_date,s.need_date),a.machine_id,a.production_minutes
 FROM mrp_planned_suggestions s LEFT JOIN mrp_machine_allocations a ON a.suggestion_code=s.code AND a.enterprise_id=s.enterprise_id
 WHERE s.plan_code=$1 AND s.enterprise_id=$2 AND s.order_type='FABRICACAO' AND NOT EXISTS(SELECT 1 FROM mrp_machine_allocation_slots sl WHERE sl.suggestion_code=s.code)
 AND NOT EXISTS(SELECT 1 FROM planned_orders po WHERE po.enterprise_id=$2 AND (po.mrp_suggestion_code=s.code OR po.order_number=s.order_number) AND po.is_active AND po.status<>'CANCELLED')
 ) SELECT o.id,o.item_code,o.quantity,o.day,mr.id,mt.id,o.minutes::double precision
 FROM orders o
 -- COALESCE nas duas pontas: roteiro sem mascara tem a coluna NULA, e NULL = ''
 -- nao e verdadeiro — era assim que o roteiro ficava invisivel para o CRP. Numa
 -- base onde NENHUM roteiro tem mascara (o caso de quem nao usa item
 -- configurado), o calculo de capacidade nao encontrava carga nenhuma e
 -- devolvia todas as ordens como "sem carga".
 LEFT JOIN LATERAL (SELECT id FROM manufacturing_routes r WHERE r.item_code=o.item_code AND r.enterprise_id=$2 AND r.is_standard AND r.is_active AND r.situation='APROVADA' AND (COALESCE(r.mask,'')=COALESCE(o.mask,'') OR COALESCE(r.mask,'')='') AND (r.valid_from IS NULL OR r.valid_from<=o.day) AND (r.valid_to IS NULL OR r.valid_to>=o.day) ORDER BY (COALESCE(r.mask,'')=COALESCE(o.mask,'')) DESC,r.id DESC LIMIT 1) mr ON TRUE
 LEFT JOIN machines m ON m.id=o.machine_id AND m.enterprise_id=$2 AND m.is_active
 LEFT JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.enterprise_id=$2 AND mt.is_active
 ORDER BY o.day,o.id`, plan, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.PlannedOrderRow{}
	for rows.Next() {
		var v domainrepo.PlannedOrderRow
		var minutes *float64
		if err := rows.Scan(&v.ID, &v.ItemCode, &v.Quantity, &v.PlannedDate, &v.RouteID, &v.MachineWorkCenterID, &minutes); err != nil {
			return nil, err
		}
		if minutes != nil {
			v.MachineHours = *minutes / 60
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *CRPRepositorySQLC) ReplaceRequirements(ctx context.Context, plan int64, reqs []*entity.CapacityRequirement) error {
	if r.pool == nil {
		if err := r.DeleteByPlan(ctx, plan); err != nil {
			return err
		}
		for _, v := range reqs {
			if _, err := r.UpsertRequirement(ctx, v); err != nil {
				return err
			}
		}
		return nil
	}
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ok bool
	if err = tx.QueryRow(ctx, `SELECT TRUE FROM production_plans WHERE code=$1 AND enterprise_id=$2 FOR UPDATE`, plan, e).Scan(&ok); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM capacity_requirements WHERE plan_code=$1 AND EXISTS(SELECT 1 FROM production_plans p WHERE p.code=$1 AND p.enterprise_id=$2)`, plan, e); err != nil {
		return err
	}
	for _, v := range reqs {
		_, err = tx.Exec(ctx, `INSERT INTO capacity_requirements(plan_code,work_center_id,req_date,required_hours,available_hours)
 SELECT $1,mt.id,$3,$4,$5 FROM machine_types mt WHERE mt.id=$2 AND mt.enterprise_id=$6`, plan, v.WorkCenterID, v.ReqDate, v.RequiredHours, v.AvailableHours, e)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *CRPRepositorySQLC) GetScheduledMachineLoads(ctx context.Context, plan int64) ([]domainrepo.ScheduledLoad, error) {
	if r.pool == nil {
		return nil, nil
	}
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT mt.id,d.day::date,
 SUM(EXTRACT(EPOCH FROM LEAST(sl.ends_at,d.day+INTERVAL '1 day')-GREATEST(sl.starts_at,d.day))/3600)::double precision
 FROM mrp_machine_allocation_slots sl
 JOIN mrp_machine_allocations a ON a.suggestion_code=sl.suggestion_code
 JOIN mrp_planned_suggestions s ON s.code=a.suggestion_code AND s.enterprise_id=a.enterprise_id
 JOIN machines m ON m.id=sl.machine_id AND m.enterprise_id=a.enterprise_id
 JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.enterprise_id=m.enterprise_id
 CROSS JOIN LATERAL generate_series(sl.starts_at::date::timestamp,(sl.ends_at-INTERVAL '1 microsecond')::date::timestamp,INTERVAL '1 day') d(day)
 WHERE s.plan_code=$1 AND s.enterprise_id=$2
 AND NOT EXISTS(SELECT 1 FROM planned_orders po WHERE po.enterprise_id=$2 AND po.mrp_suggestion_code=s.code AND (NOT po.is_active OR po.status='CANCELLED'))
 GROUP BY mt.id,d.day ORDER BY d.day,mt.id`, plan, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.ScheduledLoad{}
	for rows.Next() {
		var v domainrepo.ScheduledLoad
		if err := rows.Scan(&v.WorkCenterID, &v.Day, &v.Hours); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
