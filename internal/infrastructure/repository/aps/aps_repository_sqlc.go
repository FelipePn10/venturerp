package aps

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APSRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool ...*pgxpool.Pool) domainrepo.APSRepository {
	r := &APSRepositorySQLC{q: q}
	if len(pool) > 0 {
		r.pool = pool[0]
	}
	return r
}

func (r *APSRepositorySQLC) UpsertSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		row := r.pool.QueryRow(ctx, `INSERT INTO production_sequences(production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,enterprise_id)
			SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE EXISTS(SELECT 1 FROM production_orders WHERE id=$1 AND enterprise_id=$9)
			RETURNING id,production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,created_at,updated_at`, seq.ProductionOrderID, seq.OperationID, seq.WorkCenterID, seq.MachineID, seq.SequencePosition, seq.ScheduledStart, seq.ScheduledEnd, string(seq.Status), enterpriseID)
		return scanTenantSequence(row)
	}
	row, err := r.q.InsertProductionSequence(ctx, sqlc.InsertProductionSequenceParams{
		ProductionOrderID: seq.ProductionOrderID,
		OperationID:       pgutil.ToPgInt8Ptr(seq.OperationID),
		WorkCenterID:      seq.WorkCenterID,
		SequencePosition:  int32(seq.SequencePosition),
		ScheduledStart:    pgutil.ToPgTimestamptz(seq.ScheduledStart),
		ScheduledEnd:      pgutil.ToPgTimestamptz(seq.ScheduledEnd),
		Status:            string(seq.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("inserting sequence: %w", err)
	}
	return seqRowToEntity(row), nil
}

func (r *APSRepositorySQLC) GetSequence(ctx context.Context, id int64) (*entity.ProductionSequence, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		return scanTenantSequence(r.pool.QueryRow(ctx, `SELECT id,production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,created_at,updated_at FROM production_sequences WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID))
	}
	row, err := r.q.GetProductionSequence(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting sequence %d: %w", id, err)
	}
	return seqRowToEntity(row), nil
}

func (r *APSRepositorySQLC) UpdateSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		return scanTenantSequence(r.pool.QueryRow(ctx, `UPDATE production_sequences SET work_center_id=$2,machine_id=$3,scheduled_start=$4,scheduled_end=$5,updated_at=NOW() WHERE id=$1 AND enterprise_id=$6 RETURNING id,production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,created_at,updated_at`, seq.ID, seq.WorkCenterID, seq.MachineID, seq.ScheduledStart, seq.ScheduledEnd, enterpriseID))
	}
	row, err := r.q.UpdateProductionSequence(ctx, sqlc.UpdateProductionSequenceParams{
		ID:             seq.ID,
		WorkCenterID:   seq.WorkCenterID,
		ScheduledStart: pgutil.ToPgTimestamptz(seq.ScheduledStart),
		ScheduledEnd:   pgutil.ToPgTimestamptz(seq.ScheduledEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("updating sequence %d: %w", seq.ID, err)
	}
	return seqRowToEntity(row), nil
}

func (r *APSRepositorySQLC) ListByOrder(ctx context.Context, orderID int64) ([]*entity.ProductionSequence, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := r.pool.Query(ctx, `SELECT id,production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,created_at,updated_at FROM production_sequences WHERE production_order_id=$1 AND enterprise_id=$2 ORDER BY sequence_position`, orderID, enterpriseID)
		if err != nil {
			return nil, err
		}
		return scanTenantSequences(rows)
	}
	rows, err := r.q.ListSequencesByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing sequences for order %d: %w", orderID, err)
	}
	return seqSlice(rows), nil
}

func (r *APSRepositorySQLC) ListByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.ProductionSequence, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := r.pool.Query(ctx, `SELECT id,production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,created_at,updated_at FROM production_sequences WHERE enterprise_id=$1 AND work_center_id=$2 AND scheduled_start<$4 AND scheduled_end>$3 ORDER BY scheduled_start`, enterpriseID, workCenterID, from, to)
		if err != nil {
			return nil, err
		}
		return scanTenantSequences(rows)
	}
	rows, err := r.q.ListSequencesByWorkCenter(ctx, workCenterID,
		pgutil.ToPgTimestamptz(from), pgutil.ToPgTimestamptz(to))
	if err != nil {
		return nil, fmt.Errorf("listing sequences for work center %d: %w", workCenterID, err)
	}
	return seqSlice(rows), nil
}

func (r *APSRepositorySQLC) DeleteByOrder(ctx context.Context, orderID int64) error {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return err
		}
		_, err = r.pool.Exec(ctx, `DELETE FROM production_sequences WHERE production_order_id=$1 AND enterprise_id=$2`, orderID, enterpriseID)
		return err
	}
	return r.q.DeleteSequencesByOrder(ctx, orderID)
}

func scanTenantSequence(row pgx.Row) (*entity.ProductionSequence, error) {
	v := &entity.ProductionSequence{}
	var status string
	if err := row.Scan(&v.ID, &v.ProductionOrderID, &v.OperationID, &v.WorkCenterID, &v.MachineID, &v.SequencePosition, &v.ScheduledStart, &v.ScheduledEnd, &status, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scanning tenant sequence: %w", err)
	}
	v.Status = entity.SequenceStatus(status)
	return v, nil
}
func scanTenantSequences(rows pgx.Rows) ([]*entity.ProductionSequence, error) {
	defer rows.Close()
	out := []*entity.ProductionSequence{}
	for rows.Next() {
		v, err := scanTenantSequence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *APSRepositorySQLC) GetOpenProductionOrders(ctx context.Context) ([]domainrepo.OrderRow, error) {
	rows, err := r.q.GetOpenProductionOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching open production orders: %w", err)
	}
	out := make([]domainrepo.OrderRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domainrepo.OrderRow{
			ID:          row.ID,
			Priority:    int(row.Priority),
			PlannedDate: pgutil.FromPgTimestamptz(row.PlannedDate),
		})
	}
	return out, nil
}

func (r *APSRepositorySQLC) GetOrderOperations(ctx context.Context, orderID int64) ([]domainrepo.OpRow, error) {
	rows, err := r.q.GetOrderOperations(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("fetching operations for order %d: %w", orderID, err)
	}
	out := make([]domainrepo.OpRow, 0, len(rows))
	for _, row := range rows {
		op := domainrepo.OpRow{
			ID:           row.ID,
			Sequence:     int(row.Sequence),
			PlannedHours: pgutil.FromPgNumericToFloat64(row.PlannedHours),
			SetupHours:   pgutil.FromPgNumericToFloat64(row.SetupHours),
		}
		if row.WorkCenterID.Valid {
			v := row.WorkCenterID.Int64
			op.WorkCenterID = &v
		}
		out = append(out, op)
	}
	return out, nil
}

func (r *APSRepositorySQLC) GetWorkCenterCapacity(ctx context.Context, workCenterID int64) (float64, error) {
	return r.q.GetMachineAvailableHours(ctx, workCenterID)
}

// ─── monthly schedule board (Gantt) ───────────────────────────────────────────

func (r *APSRepositorySQLC) ListScheduledBars(ctx context.Context, from, to time.Time) ([]*entity.GanttBar, error) {
	rows, err := r.q.ListGanttScheduledBars(ctx, pgutil.ToPgTimestamptz(from), pgutil.ToPgTimestamptz(to))
	if err != nil {
		return nil, fmt.Errorf("listing scheduled bars: %w", err)
	}
	out := make([]*entity.GanttBar, 0, len(rows))
	for _, row := range rows {
		start := pgutil.FromPgTimestamptz(row.ScheduledStart)
		end := pgutil.FromPgTimestamptz(row.ScheduledEnd)
		bar := &entity.GanttBar{
			SequenceID:        row.ID,
			ProductionOrderID: row.ProductionOrderID,
			OrderNumber:       row.OrderNumber,
			ItemCode:          row.ItemCode,
			Mask:              row.Mask,
			WorkCenterID:      row.WorkCenterID,
			WorkCenterName:    row.WorkCenterName,
			OperationName:     row.OperationName,
			SequencePosition:  int(row.SequencePosition),
			Start:             start,
			End:               end,
			DurationHours:     end.Sub(start).Hours(),
			Status:            row.Status,
			Priority:          row.Priority,
			IsFallback:        false,
		}
		if row.OperationID.Valid {
			v := row.OperationID.Int64
			bar.OperationID = &v
		}
		bar.PercentComplete = scheduledPercent(
			row.Status,
			pgutil.FromPgNumericToFloat64(row.OpActualHours),
			pgutil.FromPgNumericToFloat64(row.OpPlannedHours),
			pgutil.FromPgNumericToFloat64(row.ProducedQty),
			pgutil.FromPgNumericToFloat64(row.PlannedQty),
		)
		out = append(out, bar)
	}
	return out, nil
}

func (r *APSRepositorySQLC) ListFallbackBars(ctx context.Context, from, to time.Time) ([]*entity.GanttBar, error) {
	rows, err := r.q.ListGanttFallbackBars(ctx, pgutil.ToPgDate(from), pgutil.ToPgDate(to))
	if err != nil {
		return nil, fmt.Errorf("listing fallback bars: %w", err)
	}
	out := make([]*entity.GanttBar, 0, len(rows))
	for _, row := range rows {
		startDate := pgutil.FromPgDate(row.StartDate)
		endDate := pgutil.FromPgDate(row.EndDate)
		if startDate.IsZero() {
			startDate = endDate
		}
		if endDate.IsZero() {
			endDate = startDate
		}
		// Span the inclusive end day so a single-day order still has width.
		end := endDate.Add(24 * time.Hour)
		bar := &entity.GanttBar{
			ProductionOrderID: row.ID,
			OrderNumber:       row.OrderNumber,
			ItemCode:          row.ItemCode,
			Mask:              row.Mask,
			WorkCenterID:      0,
			WorkCenterName:    "",
			Start:             startDate,
			End:               end,
			DurationHours:     end.Sub(startDate).Hours(),
			Status:            row.Status,
			Priority:          row.Priority,
			IsFallback:        true,
			PercentComplete: percent(
				pgutil.FromPgNumericToFloat64(row.ProducedQty),
				pgutil.FromPgNumericToFloat64(row.PlannedQty),
			),
		}
		out = append(out, bar)
	}
	return out, nil
}

func (r *APSRepositorySQLC) ListResourceLoad(ctx context.Context, from, to time.Time) ([]*entity.GanttResourceLoad, error) {
	rows, err := r.q.ListGanttResourceLoad(ctx, pgutil.ToPgDate(from), pgutil.ToPgDate(to))
	if err != nil {
		return nil, fmt.Errorf("listing resource load: %w", err)
	}
	out := make([]*entity.GanttResourceLoad, 0, len(rows))
	for _, row := range rows {
		req := pgutil.FromPgNumericToFloat64(row.RequiredHours)
		avail := pgutil.FromPgNumericToFloat64(row.AvailableHours)
		loadPct := 0.0
		if avail > 0 {
			loadPct = req / avail * 100
		}
		out = append(out, &entity.GanttResourceLoad{
			WorkCenterID:   row.WorkCenterID,
			Date:           pgutil.FromPgDate(row.ReqDate),
			RequiredHours:  req,
			AvailableHours: avail,
			LoadPct:        loadPct,
			IsOverloaded:   loadPct > 100,
		})
	}
	return out, nil
}

func (r *APSRepositorySQLC) ListDependencies(ctx context.Context, from, to time.Time) ([]*entity.GanttDependency, error) {
	rows, err := r.q.ListGanttDependencies(ctx, pgutil.ToPgTimestamptz(from), pgutil.ToPgTimestamptz(to))
	if err != nil {
		return nil, fmt.Errorf("listing dependencies: %w", err)
	}
	return depSlice(rows), nil
}

func (r *APSRepositorySQLC) ListOrderDependencies(ctx context.Context, orderID int64) ([]*entity.GanttDependency, error) {
	rows, err := r.q.ListGanttOrderDependencies(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing dependencies for order %d: %w", orderID, err)
	}
	return depSlice(rows), nil
}

func depSlice(rows []sqlc.DBGanttDependency) []*entity.GanttDependency {
	out := make([]*entity.GanttDependency, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.GanttDependency{
			FromSequenceID: row.FromSeq,
			ToSequenceID:   row.ToSeq,
			OverlapPct:     pgutil.FromPgNumericToFloat64(row.OverlapPct),
			Implicit:       false,
		})
	}
	return out
}

// scheduledPercent estimates how complete a sequenced operation bar is: prefer the
// operation's actual/planned hours, fall back to the order's produced/planned
// quantity, and treat a DONE sequence as fully complete.
func scheduledPercent(status string, actualHrs, plannedHrs, producedQty, plannedQty float64) float64 {
	if status == "DONE" {
		return 100
	}
	if plannedHrs > 0 {
		return clampPct(actualHrs / plannedHrs * 100)
	}
	return percent(producedQty, plannedQty)
}

func percent(done, planned float64) float64 {
	if planned <= 0 {
		return 0
	}
	return clampPct(done / planned * 100)
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func seqRowToEntity(row sqlc.DBProductionSequence) *entity.ProductionSequence {
	e := &entity.ProductionSequence{
		ID:                row.ID,
		ProductionOrderID: row.ProductionOrderID,
		WorkCenterID:      row.WorkCenterID,
		SequencePosition:  int(row.SequencePosition),
		ScheduledStart:    pgutil.FromPgTimestamptz(row.ScheduledStart),
		ScheduledEnd:      pgutil.FromPgTimestamptz(row.ScheduledEnd),
		Status:            entity.SequenceStatus(row.Status),
		CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:         pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
	if row.OperationID.Valid {
		v := row.OperationID.Int64
		e.OperationID = &v
	}
	return e
}

func seqSlice(rows []sqlc.DBProductionSequence) []*entity.ProductionSequence {
	out := make([]*entity.ProductionSequence, 0, len(rows))
	for _, row := range rows {
		out = append(out, seqRowToEntity(row))
	}
	return out
}
