package maintenance

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MaintenanceRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool ...*pgxpool.Pool) domainrepo.MaintenanceRepository {
	r := &MaintenanceRepositorySQLC{q: q}
	if len(pool) > 0 {
		r.pool = pool[0]
	}
	return r
}

func (r *MaintenanceRepositorySQLC) requireMachine(ctx context.Context, machineID int64) error {
	if r.pool == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var exists bool
	if err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM machines WHERE id=$1 AND enterprise_id=$2)`, machineID, enterpriseID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errorsuc.NewNotFoundError(fmt.Sprintf("máquina %d não encontrada na empresa autenticada", machineID))
	}
	return nil
}

func (r *MaintenanceRepositorySQLC) requireWorkCenter(ctx context.Context, workCenterID *int64) error {
	if r.pool == nil || workCenterID == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var exists bool
	if err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM machine_types WHERE id=$1 AND enterprise_id=$2)`, *workCenterID, enterpriseID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errorsuc.NewNotFoundError(fmt.Sprintf("centro de trabalho %d não encontrado na empresa autenticada", *workCenterID))
	}
	return nil
}

// ─── plans ────────────────────────────────────────────────────────────────────

func (r *MaintenanceRepositorySQLC) CreatePlan(ctx context.Context, p *entity.MaintenancePlan) (*entity.MaintenancePlan, error) {
	if err := r.requireMachine(ctx, p.MachineID); err != nil {
		return nil, err
	}
	if err := r.requireWorkCenter(ctx, p.WorkCenterID); err != nil {
		return nil, err
	}
	nextSched := pgutil.ToPgTimestamptz(time.Now().AddDate(0, 0, p.FrequencyDays))
	if p.NextScheduledAt != nil {
		nextSched = pgutil.ToPgTimestamptz(*p.NextScheduledAt)
	}
	row, err := r.q.CreateMaintenancePlan(ctx, sqlc.CreateMaintenancePlanParams{
		MachineID:       p.MachineID,
		WorkCenterID:    pgutil.ToPgInt8Ptr(p.WorkCenterID),
		Description:     p.Description,
		Frequency:       sqlc.MaintenanceFrequencyEnum(p.Frequency),
		FrequencyDays:   int32(p.FrequencyDays),
		EstimatedHours:  p.EstimatedHours,
		NextScheduledAt: nextSched,
		CreatedBy:       pgutil.ToPgUUID(p.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating maintenance plan: %w", err)
	}
	return planRowToEntity(row), nil
}

func (r *MaintenanceRepositorySQLC) UpdatePlan(ctx context.Context, p *entity.MaintenancePlan) (*entity.MaintenancePlan, error) {
	if _, err := r.GetPlanByID(ctx, p.ID); err != nil {
		return nil, err
	}
	if err := r.requireWorkCenter(ctx, p.WorkCenterID); err != nil {
		return nil, err
	}
	nextSched := pgutil.ToPgTimestamptz(time.Now().AddDate(0, 0, p.FrequencyDays))
	if p.NextScheduledAt != nil {
		nextSched = pgutil.ToPgTimestamptz(*p.NextScheduledAt)
	}
	row, err := r.q.UpdateMaintenancePlan(ctx, sqlc.UpdateMaintenancePlanParams{
		ID:              p.ID,
		Description:     p.Description,
		Frequency:       sqlc.MaintenanceFrequencyEnum(p.Frequency),
		FrequencyDays:   int32(p.FrequencyDays),
		EstimatedHours:  p.EstimatedHours,
		NextScheduledAt: nextSched,
	})
	if err != nil {
		return nil, fmt.Errorf("updating maintenance plan: %w", err)
	}
	return planRowToEntity(row), nil
}

func (r *MaintenanceRepositorySQLC) GetPlanByID(ctx context.Context, id int64) (*entity.MaintenancePlan, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		var allowed bool
		if err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM maintenance_plans p JOIN machines m ON m.id=p.machine_id WHERE p.id=$1 AND m.enterprise_id=$2)`, id, enterpriseID).Scan(&allowed); err != nil {
			return nil, err
		}
		if !allowed {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("plano de manutenção %d não encontrado", id))
		}
	}
	row, err := r.q.GetMaintenancePlanByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching maintenance plan %d: %w", id, err)
	}
	return planRowToEntity(row), nil
}

func (r *MaintenanceRepositorySQLC) ListPlans(ctx context.Context, onlyActive bool) ([]*entity.MaintenancePlan, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := r.pool.Query(ctx, `SELECT p.id,p.code,p.machine_id,p.work_center_id,p.description,p.frequency,p.frequency_days,p.estimated_hours,p.last_executed_at,p.next_scheduled_at,p.is_active,p.created_at,p.updated_at,p.created_by FROM maintenance_plans p JOIN machines m ON m.id=p.machine_id WHERE m.enterprise_id=$1 AND (NOT $2 OR p.is_active) ORDER BY p.machine_id,p.next_scheduled_at`, enterpriseID, onlyActive)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []*entity.MaintenancePlan{}
		for rows.Next() {
			var row sqlc.DBMaintenancePlan
			if err := rows.Scan(&row.ID, &row.Code, &row.MachineID, &row.WorkCenterID, &row.Description, &row.Frequency, &row.FrequencyDays, &row.EstimatedHours, &row.LastExecutedAt, &row.NextScheduledAt, &row.IsActive, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy); err != nil {
				return nil, err
			}
			out = append(out, planRowToEntity(row))
		}
		return out, rows.Err()
	}
	rows, err := r.q.ListMaintenancePlans(ctx, &onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing maintenance plans: %w", err)
	}
	return planSlice(rows), nil
}

func (r *MaintenanceRepositorySQLC) ListPlansByMachine(ctx context.Context, machineID int64) ([]*entity.MaintenancePlan, error) {
	if err := r.requireMachine(ctx, machineID); err != nil {
		return nil, err
	}
	rows, err := r.q.ListMaintenancePlansByMachine(ctx, machineID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os planos da máquina %d: %w", machineID, err)
	}
	return planSlice(rows), nil
}

func (r *MaintenanceRepositorySQLC) DeactivatePlan(ctx context.Context, id int64) error {
	if _, err := r.GetPlanByID(ctx, id); err != nil {
		return err
	}
	return r.q.DeactivateMaintenancePlan(ctx, id)
}

// ─── orders ───────────────────────────────────────────────────────────────────

func (r *MaintenanceRepositorySQLC) CreateOrder(ctx context.Context, o *entity.MaintenanceOrder) (*entity.MaintenanceOrder, error) {
	if _, err := r.GetPlanByID(ctx, o.PlanID); err != nil {
		return nil, err
	}
	if err := r.requireWorkCenter(ctx, o.WorkCenterID); err != nil {
		return nil, err
	}
	row, err := r.q.CreateMaintenanceOrder(ctx, sqlc.CreateMaintenanceOrderParams{
		PlanID:         o.PlanID,
		MachineID:      pgutil.ToPgInt8Ptr(o.MachineID),
		WorkCenterID:   pgutil.ToPgInt8Ptr(o.WorkCenterID),
		ScheduledDate:  pgutil.ToPgDate(o.ScheduledDate),
		EstimatedHours: o.EstimatedHours,
	})
	if err != nil {
		return nil, fmt.Errorf("creating maintenance order: %w", err)
	}
	return orderRowToEntity(row), nil
}

func (r *MaintenanceRepositorySQLC) UpdateOrder(ctx context.Context, o *entity.MaintenanceOrder) (*entity.MaintenanceOrder, error) {
	if _, err := r.GetOrderByID(ctx, o.ID); err != nil {
		return nil, err
	}
	actualH := pgtype.Float8{}
	if o.ActualHours != nil {
		actualH = pgtype.Float8{Float64: *o.ActualHours, Valid: true}
	}
	startedAt := pgtype.Timestamptz{}
	if o.StartedAt != nil {
		startedAt = pgutil.ToPgTimestamptz(*o.StartedAt)
	}
	completedAt := pgtype.Timestamptz{}
	if o.CompletedAt != nil {
		completedAt = pgutil.ToPgTimestamptz(*o.CompletedAt)
	}
	row, err := r.q.UpdateMaintenanceOrder(ctx, sqlc.UpdateMaintenanceOrderParams{
		ID:          o.ID,
		Status:      sqlc.MaintenanceOrderStatusEnum(o.Status),
		ActualHours: actualH,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Notes:       pgutil.ToPgTextFromPtr(o.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("updating maintenance order %d: %w", o.ID, err)
	}
	return orderRowToEntity(row), nil
}

func (r *MaintenanceRepositorySQLC) GetOrderByID(ctx context.Context, id int64) (*entity.MaintenanceOrder, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		var allowed bool
		if err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM maintenance_orders o JOIN maintenance_plans p ON p.id=o.plan_id JOIN machines m ON m.id=p.machine_id WHERE o.id=$1 AND m.enterprise_id=$2)`, id, enterpriseID).Scan(&allowed); err != nil {
			return nil, err
		}
		if !allowed {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ordem de manutenção %d não encontrada", id))
		}
	}
	row, err := r.q.GetMaintenanceOrderByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching maintenance order %d: %w", id, err)
	}
	return orderRowToEntity(row), nil
}

func (r *MaintenanceRepositorySQLC) ListOrdersByPlan(ctx context.Context, planID int64) ([]*entity.MaintenanceOrder, error) {
	if _, err := r.GetPlanByID(ctx, planID); err != nil {
		return nil, err
	}
	rows, err := r.q.ListMaintenanceOrdersByPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as ordens do plano %d: %w", planID, err)
	}
	return orderSlice(rows), nil
}

func (r *MaintenanceRepositorySQLC) ListOrdersByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.MaintenanceOrder, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := r.pool.Query(ctx, `SELECT o.id,o.plan_id,o.machine_id,o.work_center_id,o.scheduled_date,o.estimated_hours,o.actual_hours,o.status,o.started_at,o.completed_at,o.notes,o.is_active,o.created_at,o.updated_at FROM maintenance_orders o JOIN maintenance_plans p ON p.id=o.plan_id JOIN machines m ON m.id=p.machine_id WHERE m.enterprise_id=$1 AND o.work_center_id=$2 AND o.scheduled_date BETWEEN $3 AND $4 AND o.is_active ORDER BY o.scheduled_date`, enterpriseID, workCenterID, from, to)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []*entity.MaintenanceOrder{}
		for rows.Next() {
			var row sqlc.DBMaintenanceOrder
			if err := rows.Scan(&row.ID, &row.PlanID, &row.MachineID, &row.WorkCenterID, &row.ScheduledDate, &row.EstimatedHours, &row.ActualHours, &row.Status, &row.StartedAt, &row.CompletedAt, &row.Notes, &row.IsActive, &row.CreatedAt, &row.UpdatedAt); err != nil {
				return nil, err
			}
			out = append(out, orderRowToEntity(row))
		}
		return out, rows.Err()
	}
	rows, err := r.q.ListMaintenanceOrdersByWorkCenter(ctx, workCenterID,
		pgutil.ToPgDate(from), pgutil.ToPgDate(to))
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as ordens do centro de trabalho %d: %w", workCenterID, err)
	}
	return orderSlice(rows), nil
}

func (r *MaintenanceRepositorySQLC) ExistsOrderForPlanAndDate(ctx context.Context, planID int64, date time.Time) (bool, error) {
	if _, err := r.GetPlanByID(ctx, planID); err != nil {
		return false, err
	}
	return r.q.ExistsOrderForPlanAndDate(ctx, planID, date)
}

func (r *MaintenanceRepositorySQLC) GetBlockedHours(ctx context.Context, workCenterID int64, date time.Time) (float64, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return 0, err
		}
		var hours float64
		err = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(o.estimated_hours),0)::float8 FROM maintenance_orders o JOIN maintenance_plans p ON p.id=o.plan_id JOIN machines m ON m.id=p.machine_id WHERE m.enterprise_id=$1 AND o.work_center_id=$2 AND o.scheduled_date=$3 AND o.status IN ('PLANNED','IN_PROGRESS') AND o.is_active`, enterpriseID, workCenterID, date).Scan(&hours)
		return hours, err
	}
	return r.q.GetBlockedHoursOnDate(ctx, workCenterID, date)
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func planRowToEntity(row sqlc.DBMaintenancePlan) *entity.MaintenancePlan {
	p := &entity.MaintenancePlan{
		ID:             row.ID,
		Code:           row.Code,
		MachineID:      row.MachineID,
		Description:    row.Description,
		Frequency:      entity.Frequency(row.Frequency),
		FrequencyDays:  int(row.FrequencyDays),
		EstimatedHours: row.EstimatedHours,
		IsActive:       row.IsActive,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:      pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:      pgutil.FromPgUUID(row.CreatedBy),
	}
	if row.WorkCenterID.Valid {
		v := row.WorkCenterID.Int64
		p.WorkCenterID = &v
	}
	if row.LastExecutedAt.Valid {
		t := pgutil.FromPgTimestamptz(row.LastExecutedAt)
		p.LastExecutedAt = &t
	}
	if row.NextScheduledAt.Valid {
		t := pgutil.FromPgTimestamptz(row.NextScheduledAt)
		p.NextScheduledAt = &t
	}
	return p
}

func planSlice(rows []sqlc.DBMaintenancePlan) []*entity.MaintenancePlan {
	out := make([]*entity.MaintenancePlan, 0, len(rows))
	for _, row := range rows {
		out = append(out, planRowToEntity(row))
	}
	return out
}

func orderRowToEntity(row sqlc.DBMaintenanceOrder) *entity.MaintenanceOrder {
	o := &entity.MaintenanceOrder{
		ID:             row.ID,
		PlanID:         row.PlanID,
		ScheduledDate:  pgutil.FromPgDate(row.ScheduledDate),
		EstimatedHours: row.EstimatedHours,
		Status:         entity.OrderStatus(row.Status),
		IsActive:       row.IsActive,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:      pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
	if row.MachineID.Valid {
		v := row.MachineID.Int64
		o.MachineID = &v
	}
	if row.WorkCenterID.Valid {
		v := row.WorkCenterID.Int64
		o.WorkCenterID = &v
	}
	if row.ActualHours.Valid {
		v := row.ActualHours.Float64
		o.ActualHours = &v
	}
	if row.StartedAt.Valid {
		t := pgutil.FromPgTimestamptz(row.StartedAt)
		o.StartedAt = &t
	}
	if row.CompletedAt.Valid {
		t := pgutil.FromPgTimestamptz(row.CompletedAt)
		o.CompletedAt = &t
	}
	if row.Notes.Valid {
		v := row.Notes.String
		o.Notes = &v
	}
	return o
}

func orderSlice(rows []sqlc.DBMaintenanceOrder) []*entity.MaintenanceOrder {
	out := make([]*entity.MaintenanceOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, orderRowToEntity(row))
	}
	return out
}
