package delivery_reschedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *DeliveryRescheduleRepositorySQLC) Create(
	ctx context.Context,
	res *entity.DeliveryReschedule,
) (*entity.DeliveryReschedule, error) {

	row, err := r.q.CreateDeliveryReschedule(
		ctx,
		sqlc.CreateDeliveryRescheduleParams{
			Code:           res.Code,
			SalesOrderCode: res.SalesOrderCode,
			ItemCode:       int64(res.ItemCode),
			OldDate:        pgutil.ToPgDate(res.OldDate),
			NewDate:        pgutil.ToPgDate(res.NewDate),

			Reason: pgutil.ToPgTextFromPtr(res.Reason),

			CreatedBy: pgutil.ToPgUUID(res.CreatedBy),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating delivery reschedule: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.DeliveryReschedule, error) {

	row, err := r.q.GetDeliveryRescheduleByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("delivery reschedule %d not found", code)
		}
		return nil, fmt.Errorf("fetching delivery reschedule: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByOrder(
	ctx context.Context,
	salesOrderCode int64,
) ([]*entity.DeliveryReschedule, error) {

	rows, err := r.q.ListReschedulesByOrder(ctx, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by order: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByItem(
	ctx context.Context,
	itemCode valueobject.ItemCode,
) ([]*entity.DeliveryReschedule, error) {

	rows, err := r.q.ListReschedulesByItem(ctx, int64(itemCode))
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by item: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) Delete(
	ctx context.Context,
	id int64,
) error {
	return r.q.DeleteDeliveryReschedule(ctx, id)
}

func rowToEntity(row sqlc.DeliveryReschedule) *entity.DeliveryReschedule {
	e := &entity.DeliveryReschedule{
		Code:           row.Code,
		SalesOrderCode: row.SalesOrderCode,
		ItemCode:       valueobject.ItemCode(row.ItemCode),
		OldDate:        pgutil.FromPgDate(row.OldDate),
		NewDate:        pgutil.FromPgDate(row.NewDate),
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:      pgutil.FromPgUUID(row.CreatedBy),
	}

	e.Reason = pgutil.FromPgTextPtr(row.Reason)

	return e
}

func rowsToEntities(rows []sqlc.DeliveryReschedule) []*entity.DeliveryReschedule {
	out := make([]*entity.DeliveryReschedule, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}
