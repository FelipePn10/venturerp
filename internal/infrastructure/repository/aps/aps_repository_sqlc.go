package aps

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type APSRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.APSRepository {
	return &APSRepositorySQLC{q: q}
}

func (r *APSRepositorySQLC) UpsertSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error) {
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

func (r *APSRepositorySQLC) ListByOrder(ctx context.Context, orderID int64) ([]*entity.ProductionSequence, error) {
	rows, err := r.q.ListSequencesByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing sequences for order %d: %w", orderID, err)
	}
	return seqSlice(rows), nil
}

func (r *APSRepositorySQLC) ListByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.ProductionSequence, error) {
	rows, err := r.q.ListSequencesByWorkCenter(ctx, workCenterID,
		pgutil.ToPgTimestamptz(from), pgutil.ToPgTimestamptz(to))
	if err != nil {
		return nil, fmt.Errorf("listing sequences for work center %d: %w", workCenterID, err)
	}
	return seqSlice(rows), nil
}

func (r *APSRepositorySQLC) DeleteByOrder(ctx context.Context, orderID int64) error {
	return r.q.DeleteSequencesByOrder(ctx, orderID)
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
