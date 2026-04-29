package delivery_reschedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *DeliveryRescheduleRepositorySQLC) Create(ctx context.Context, res *entity.DeliveryReschedule) (*entity.DeliveryReschedule, error) {
	row, err := r.q.CreateDeliveryReschedule(ctx, sqlc.CreateDeliveryRescheduleParams{
		Code:           res.Code,
		SalesOrderCode: res.SalesOrderCode,
		ItemCode:       int64(res.ItemCode),
		OldDate:        res.OldDate,
		NewDate:        res.NewDate,
		Reason:         toNullString(res.Reason),
		CreatedBy:      res.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("creating delivery reschedule: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.DeliveryReschedule, error) {
	row, err := r.q.GetDeliveryRescheduleByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("delivery reschedule %d not found", code)
		}
		return nil, fmt.Errorf("fetching delivery reschedule: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByOrder(ctx context.Context, salesOrderCode int64) ([]*entity.DeliveryReschedule, error) {
	rows, err := r.q.ListReschedulesByOrder(ctx, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by salesOrderCode: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByItem(ctx context.Context, itemCode valueobject.ItemCode) ([]*entity.DeliveryReschedule, error) {
	rows, err := r.q.ListReschedulesByItem(ctx, int64(itemCode))
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by item: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) Delete(ctx context.Context, id int64) error {
	return r.q.DeleteDeliveryReschedule(ctx, id)
}

func rowToEntity(row sqlc.DeliveryReschedule) *entity.DeliveryReschedule {
	e := &entity.DeliveryReschedule{
		Code:           row.Code,
		SalesOrderCode: row.SalesOrderCode,
		ItemCode:       valueobject.ItemCode(row.ItemCode),
		OldDate:        row.OldDate,
		NewDate:        row.NewDate,
		CreatedAt:      row.CreatedAt,
		CreatedBy:      row.CreatedBy,
	}
	if row.Reason.Valid {
		v := row.Reason.String
		e.Reason = &v
	}
	return e
}

func rowsToEntities(rows []sqlc.DeliveryReschedule) []*entity.DeliveryReschedule {
	out := make([]*entity.DeliveryReschedule, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func Int64Ptr(v int64) *int64 {
	return &v
}
