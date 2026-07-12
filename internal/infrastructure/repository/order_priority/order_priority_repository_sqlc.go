package order_priority

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

func (r *OrderPriorityRepositorySQLC) Create(
	ctx context.Context,
	op *entity.OrderPriority,
) (*entity.OrderPriority, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.CreateOrderPriority(
		ctx,
		sqlc.CreateOrderPriorityParams{
			IntervalStart: pgutil.ToPgNumericFromFloat64(op.IntervalStart),
			IntervalEnd:   pgutil.ToPgNumericFromFloat64(op.IntervalEnd),
			Priority:      op.Priority,
			Description:   pgutil.ToPgTextFromPtr(op.Description),
			CreatedBy:     pgutil.ToPgUUID(op.CreatedBy),
			EnterpriseID:  enterpriseID,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("creating order priority: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *OrderPriorityRepositorySQLC) Update(
	ctx context.Context,
	op *entity.OrderPriority,
) (*entity.OrderPriority, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.UpdateOrderPriority(
		ctx,
		sqlc.UpdateOrderPriorityParams{
			IntervalStart: pgutil.ToPgNumericFromFloat64(op.IntervalStart),
			IntervalEnd:   pgutil.ToPgNumericFromFloat64(op.IntervalEnd),
			Priority:      op.Priority,
			Description:   pgutil.ToPgTextFromPtr(op.Description),
			Code:          op.Code,
			EnterpriseID:  enterpriseID,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("updating order priority: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *OrderPriorityRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.OrderPriority, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetOrderPriorityByCode(ctx, sqlc.GetOrderPriorityByCodeParams{Code: code, EnterpriseID: enterpriseID})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order priority %d not found", code)
		}

		return nil, fmt.Errorf("fetching order priority: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *OrderPriorityRepositorySQLC) FindByValue(
	ctx context.Context,
	value float64,
) (*entity.OrderPriority, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.FindPriorityByValue(
		ctx,
		sqlc.FindPriorityByValueParams{IntervalStart: pgutil.ToPgNumericFromFloat64(value), EnterpriseID: enterpriseID},
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("finding priority by value: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *OrderPriorityRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.OrderPriority, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListOrderPriorities(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing order priorities: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *OrderPriorityRepositorySQLC) Delete(
	ctx context.Context,
	code int64,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteOrderPriority(ctx, sqlc.DeleteOrderPriorityParams{Code: code, EnterpriseID: enterpriseID})
}

func rowToEntity(
	row sqlc.OrderPriority,
) *entity.OrderPriority {

	e := &entity.OrderPriority{
		Code:          row.Code,
		IntervalStart: pgutil.FromPgNumericToFloat64(row.IntervalStart),
		IntervalEnd:   pgutil.FromPgNumericToFloat64(row.IntervalEnd),
		Priority:      row.Priority,
		IsActive:      row.IsActive,
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:     pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:     pgutil.FromPgUUID(row.CreatedBy),
	}

	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}

	return e
}

func rowsToEntities(
	rows []sqlc.OrderPriority,
) []*entity.OrderPriority {

	out := make([]*entity.OrderPriority, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}
